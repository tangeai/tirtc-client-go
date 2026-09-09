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
	deps    map[replayDependency]struct{}
	tasks   map[*RecordingTask]struct{}
	closed  bool
	client  *Client
	device  *native.CloudStorage
}

type replayDependency interface {
	preflightClientClose() error
	detachFromClient() error
}

func (c *Client) NewReplay(deviceID string, options ReplayOptions) (*Replay, error) {
	if c == nil {
		return nil, ErrClosed
	}
	c.op.enter()
	device, err := c.openDeviceLocked(deviceID)
	if err != nil {
		c.op.leave()
		return nil, err
	}
	replay := &Replay{queue: newCallbackQueue(), options: options, client: c, device: device,
		deps: make(map[replayDependency]struct{}), tasks: make(map[*RecordingTask]struct{})}
	nativeReplay, code := device.NewReplay(native.CloudStorageReplayCallbacks{
		Dispatch: func(task func()) {
			// Keep generation checks in the Runtime task until the user handler runs.
			// Every accepted opaque task must execute, including suppressed teardown work.
			if !replay.queue.postReliable(task) {
				task()
			}
		},
		OnTime: func(value int64) {
			if replay.options.OnTimeChanged != nil {
				replay.options.OnTimeChanged(time.UnixMilli(value).UTC())
			}
		},
		OnCompleted: func() {
			logCloudStorageEvent("cloud_storage_replay_completed")
			if replay.options.OnCompleted != nil {
				replay.options.OnCompleted()
			}
		},
		OnError: func(code int32) {
			logCloudStorageResult("cloud_storage_replay", nativeError(code))
			if replay.options.OnError != nil {
				replay.options.OnError(nativeError(code))
			}
		},
		OnGap: func(value native.CloudStorageRecordingGap) {
			if replay.options.OnRecordingGap != nil {
				replay.options.OnRecordingGap(recordingGapFromNative(value))
			}
		},
	})
	if code != 0 {
		replay.queue.close()
		_ = c.closeDevice(device)
		c.op.leave()
		err := nativeError(code)
		logCloudStorageResult("cloud_storage_replay_create", err)
		return nil, err
	}
	replay.native = nativeReplay
	c.registerChildLocked(replay)
	c.op.leave()
	logCloudStorageResult("cloud_storage_replay_create", nil)
	return replay, nil
}

// enterNative never waits behind an operation that may be draining this callback.
func (r *Replay) enterNative() bool {
	if r.queue.inHandler() {
		return r.op.mu.TryLock()
	}
	r.op.enter()
	return true
}

func (r *Replay) withNative(operationName string, operation func(*native.CloudStorageReplay) int32) error {
	return r.withNativeThen(operationName, operation, nil)
}

func (r *Replay) withNativeThen(operationName string, operation func(*native.CloudStorageReplay) int32,
	afterSuccess func()) error {
	if !r.enterNative() {
		return ErrInUse
	}
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

func (r *Replay) Play(startTime, endTime time.Time) error {
	return r.PlayAt(startTime, endTime, startTime)
}

func (r *Replay) PlayAt(startTime, endTime, initialTime time.Time) error {
	if r.queue.inHandler() {
		return ErrInUse
	}
	start, end, initial := unixMilliseconds(startTime), unixMilliseconds(endTime), unixMilliseconds(initialTime)
	return r.withNative("cloud_storage_replay_play", func(handle *native.CloudStorageReplay) int32 {
		return handle.Play(start, end, initial)
	})
}

func (r *Replay) Pause() error {
	return r.withNative("cloud_storage_replay_pause", func(handle *native.CloudStorageReplay) int32 { return handle.Pause() })
}

func (r *Replay) Resume() error {
	return r.withNative("cloud_storage_replay_resume", func(handle *native.CloudStorageReplay) int32 { return handle.Resume() })
}

func (r *Replay) Seek(target time.Time) error {
	if r.queue.inHandler() {
		return ErrInUse
	}
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
	if !r.enterNative() {
		return time.Time{}, false, ErrInUse
	}
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
	if r.queue.inHandler() {
		return ErrInUse
	}
	return r.withNative("cloud_storage_replay_stop", func(handle *native.CloudStorageReplay) int32 {
		return handle.Stop()
	})
}

func (r *Replay) StartRecording(options StartRecordingOptions) (*RecordingTask, error) {
	audio := int32(-1)
	if options.AudioChannelID != nil {
		audio = int32(*options.AudioChannelID)
	}
	if !r.enterNative() {
		return nil, ErrInUse
	}
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
	taskValue := &RecordingTask{native: task, replay: r}
	r.tasks[taskValue] = struct{}{}
	r.mu.Unlock()
	logCloudStorageResult("cloud_storage_recording_start", nil)
	return taskValue, nil
}

func (r *Replay) Close() (resultErr error) {
	defer func() { logCloudStorageResult("cloud_storage_replay_dispose", resultErr) }()
	if !r.enterNative() {
		return ErrInUse
	}
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
	if len(r.deps) != 0 || len(r.tasks) != 0 {
		r.mu.Unlock()
		r.op.leave()
		return ErrInUse
	}
	handle := r.native
	r.mu.Unlock()
	if err := nativeError(handle.Stop()); err != nil {
		r.op.leave()
		return err
	}
	if err := nativeError(handle.Close()); err != nil {
		r.op.leave()
		return err
	}
	r.mu.Lock()
	r.closed = true
	r.native = nil
	device := r.device
	client := r.client
	r.device = nil
	r.client = nil
	r.mu.Unlock()
	finishNativeClose(&r.op, r.queue)
	if client != nil {
		client.unregisterChild(r)
		return client.closeDevice(device)
	}
	return nil
}

func (r *Replay) preflightClientClose() error {
	if !r.queue.replayCloseReady() {
		return ErrInUse
	}
	r.mu.Lock()
	dependencies := make([]replayDependency, 0, len(r.deps))
	for dependency := range r.deps {
		dependencies = append(dependencies, dependency)
	}
	r.mu.Unlock()
	for _, dependency := range dependencies {
		if err := dependency.preflightClientClose(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Replay) closeFromClient() error {
	r.mu.Lock()
	dependencies := make([]replayDependency, 0, len(r.deps))
	for dependency := range r.deps {
		dependencies = append(dependencies, dependency)
	}
	tasks := make([]*RecordingTask, 0, len(r.tasks))
	for task := range r.tasks {
		tasks = append(tasks, task)
	}
	r.mu.Unlock()
	for _, task := range tasks {
		if _, err := task.Stop(); err != nil {
			task.mu.Lock()
			done := task.done
			task.mu.Unlock()
			if !done {
				return err
			}
		}
	}
	for _, dependency := range dependencies {
		if err := dependency.detachFromClient(); err != nil {
			return err
		}
	}
	return r.Close()
}
