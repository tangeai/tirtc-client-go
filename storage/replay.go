package storage

import (
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type Replay struct {
	op      nativeOperationGate
	mu      sync.Mutex
	native  *native.CloudStorageReplay
	queue   *callbackQueue
	options ReplayOptions
	speed   ReplaySpeed
	deps    int
	tasks   int
	active  bool
	closed  bool
}

func (s *CloudStorage) NewReplay(options ReplayOptions) (*Replay, error) {
	s.op.enter()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.op.leave()
		return nil, ErrClosed
	}
	handle := s.native
	s.mu.Unlock()
	replay := &Replay{queue: newCallbackQueue(), options: options}
	nativeReplay, code := handle.NewReplay(native.CloudStorageReplayCallbacks{
		OnTime: func(value int64) {
			_ = replay.queue.post(func() {
				if replay.options.OnTimeChanged != nil {
					replay.options.OnTimeChanged(time.UnixMilli(value).UTC())
				}
			})
		},
		OnCompleted: func() {
			replay.mu.Lock()
			replay.active = false
			logCloudStorageEvent("cloud_storage_replay_completed")
			_ = replay.queue.postReservedTerminal(func() {
				if replay.options.OnCompleted != nil {
					replay.options.OnCompleted()
				}
			})
			replay.mu.Unlock()
		},
		OnError: func(code int32) {
			replay.mu.Lock()
			replay.active = false
			logCloudStorageResult("cloud_storage_replay", nativeError(code))
			_ = replay.queue.postReservedTerminal(func() {
				if replay.options.OnError != nil {
					replay.options.OnError(nativeError(code))
				}
			})
			replay.mu.Unlock()
		},
	})
	s.op.leave()
	if code != 0 {
		replay.queue.close()
		err := nativeError(code)
		logCloudStorageResult("cloud_storage_replay_create", err)
		return nil, err
	}
	replay.native = nativeReplay
	logCloudStorageResult("cloud_storage_replay_create", nil)
	return replay, nil
}

func (r *Replay) withNative(operationName string, operation func(*native.CloudStorageReplay) int32) error {
	return r.withNativeThen(operationName, operation, nil)
}

func (r *Replay) withNativeThen(operationName string, operation func(*native.CloudStorageReplay) int32,
	afterSuccess func()) error {
	r.op.enter()
	defer r.op.leave()
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ErrClosed
	}
	handle := r.native
	r.mu.Unlock()
	err := nativeError(operation(handle))
	if err == nil && afterSuccess != nil {
		afterSuccess()
	}
	logCloudStorageResult(operationName, err)
	return err
}

func (r *Replay) withReservedTerminalNative(operationName string,
	operation func(*native.CloudStorageReplay) int32) error {
	r.op.enter()
	defer r.op.leave()
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ErrClosed
	}
	handle := r.native
	r.mu.Unlock()
	createdReservation, reserved := r.queue.ensureTerminalReservation()
	if !reserved {
		return ErrInUse
	}
	err := nativeError(operation(handle))
	if err != nil && createdReservation {
		r.queue.releaseTerminal()
	}
	logCloudStorageResult(operationName, err)
	return err
}

func (r *Replay) Play(startTime, endTime time.Time) error {
	return r.PlayAt(startTime, endTime, startTime)
}

func (r *Replay) PlayAt(startTime, endTime, initialTime time.Time) error {
	start, end, initial := unixMilliseconds(startTime), unixMilliseconds(endTime), unixMilliseconds(initialTime)
	err := r.withReservedTerminalNative("cloud_storage_replay_play", func(handle *native.CloudStorageReplay) int32 {
		return handle.Play(start, end, initial)
	})
	if err == nil {
		r.mu.Lock()
		r.active = r.queue.terminalPending()
		r.mu.Unlock()
	}
	return err
}

func (r *Replay) Pause() error {
	return r.withNative("cloud_storage_replay_pause", func(handle *native.CloudStorageReplay) int32 { return handle.Pause() })
}

func (r *Replay) Resume() error {
	return r.withNative("cloud_storage_replay_resume", func(handle *native.CloudStorageReplay) int32 { return handle.Resume() })
}

func (r *Replay) Seek(target time.Time) error {
	value := unixMilliseconds(target)
	return r.withNative("cloud_storage_replay_seek", func(handle *native.CloudStorageReplay) int32 { return handle.SeekTo(value) })
}

func (r *Replay) SetSpeed(speed ReplaySpeed) error {
	if speed != ReplaySpeed0_125x && speed != ReplaySpeed0_25x && speed != ReplaySpeed0_5x &&
		speed != ReplaySpeed1x && speed != ReplaySpeed2x && speed != ReplaySpeed4x && speed != ReplaySpeed8x {
		return ErrInvalidArgument
	}
	return r.withNativeThen("cloud_storage_replay_set_speed", func(handle *native.CloudStorageReplay) int32 {
		return handle.SetSpeed(uint32(speed))
	}, func() {
		r.mu.Lock()
		r.speed = speed
		r.mu.Unlock()
	})
}

func (r *Replay) Speed() ReplaySpeed {
	r.mu.Lock()
	speed := r.speed
	r.mu.Unlock()
	return speed
}

func (r *Replay) CurrentTime() (time.Time, bool, error) {
	r.op.enter()
	defer r.op.leave()
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return time.Time{}, false, ErrClosed
	}
	handle := r.native
	r.mu.Unlock()
	value, present, code := handle.CurrentTime()
	if code != 0 {
		return time.Time{}, false, nativeError(code)
	}
	if !present {
		return time.Time{}, false, nil
	}
	return time.UnixMilli(value).UTC(), true, nil
}

func (r *Replay) Stop() error {
	return r.withNativeThen("cloud_storage_replay_stop", func(handle *native.CloudStorageReplay) int32 {
		return handle.Stop()
	}, func() {
		r.mu.Lock()
		r.active = false
		r.mu.Unlock()
		r.queue.releaseTerminal()
	})
}

func (r *Replay) StartRecording(options StartRecordingOptions) (*RecordingTask, error) {
	audio := int32(-1)
	if options.AudioChannelID != nil {
		audio = int32(*options.AudioChannelID)
	}
	r.op.enter()
	defer r.op.leave()
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrClosed
	}
	handle := r.native
	r.mu.Unlock()
	task, code := handle.StartRecording(int32(options.VideoChannelID), audio)
	if code != 0 {
		err := nativeError(code)
		logCloudStorageResult("cloud_storage_recording_start", err)
		return nil, err
	}
	r.mu.Lock()
	r.tasks++
	r.mu.Unlock()
	logCloudStorageResult("cloud_storage_recording_start", nil)
	return &RecordingTask{native: task, replay: r}, nil
}

func (r *Replay) Close() (resultErr error) {
	defer func() { logCloudStorageResult("cloud_storage_replay_dispose", resultErr) }()
	r.op.enter()
	if !r.queue.replayCloseReady() {
		r.op.leave()
		return ErrInUse
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		r.op.leave()
		return nil
	}
	if r.deps != 0 || r.tasks != 0 {
		r.mu.Unlock()
		r.op.leave()
		return ErrInUse
	}
	handle := r.native
	active := r.active
	r.mu.Unlock()
	if active {
		if err := nativeError(handle.Stop()); err != nil {
			r.op.leave()
			return err
		}
		r.mu.Lock()
		r.active = false
		r.mu.Unlock()
		r.queue.releaseTerminal()
	}
	if err := nativeError(handle.Close()); err != nil {
		r.op.leave()
		return err
	}
	r.mu.Lock()
	r.closed = true
	r.native = nil
	r.mu.Unlock()
	finishNativeClose(&r.op, r.queue)
	return nil
}
