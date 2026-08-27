package storage

import (
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type outputBase struct {
	op      nativeOperationGate
	mu      sync.Mutex
	queue   *callbackQueue
	state   OutputState
	bound   *Replay
	channel uint8
	closed  bool
	dropped bool
}

func newOutputBase() outputBase { return outputBase{queue: newCallbackQueue()} }

func outputStateFromNative(state uint32) OutputState {
	switch state {
	case 0:
		return OutputIdle
	case 1:
		return OutputBuffering
	case 2:
		return OutputDelivering
	case 3:
		return OutputFailed
	case 4:
		return OutputPaused
	case 5:
		return OutputCompleted
	default:
		return OutputFailed
	}
}

func postOutputState(queue *callbackQueue, state OutputState, callback func()) bool {
	if state == OutputCompleted || state == OutputFailed {
		return queue.postCritical(criticalTerminalState, callback, false)
	}
	return queue.postCritical(criticalLatestState, callback, true)
}

func postOutputError(queue *callbackQueue, callback func()) bool {
	return queue.postCritical(criticalError, callback, false)
}

func (b *outputBase) State() OutputState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func (b *outputBase) setState(state OutputState) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == state {
		return false
	}
	b.state = state
	return true
}

func (b *outputBase) sourceTime(value int64, present bool) time.Time {
	if !present {
		return time.Time{}
	}
	return time.UnixMicro(value).UTC()
}

func (b *outputBase) discontinuity(value bool) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	value = value || b.dropped
	b.dropped = false
	return value
}

func (b *outputBase) markDropped() {
	b.mu.Lock()
	b.dropped = true
	b.mu.Unlock()
}

func (b *outputBase) attach(replay *Replay, channelID uint8, operation func(*native.CloudStorageReplay) int32) error {
	if replay == nil {
		return ErrInvalidArgument
	}
	replay.op.enter()
	defer replay.op.leave()
	b.op.enter()
	defer b.op.leave()
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	if b.bound != nil {
		if b.bound == replay && b.channel == channelID {
			b.mu.Unlock()
			return nil
		}
		b.mu.Unlock()
		return ErrInUse
	}
	b.mu.Unlock()
	replay.mu.Lock()
	if replay.closed {
		replay.mu.Unlock()
		return ErrClosed
	}
	handle := replay.native
	replay.mu.Unlock()
	if err := nativeError(operation(handle)); err != nil {
		return err
	}
	b.mu.Lock()
	b.bound = replay
	b.channel = channelID
	b.mu.Unlock()
	replay.mu.Lock()
	replay.deps++
	replay.mu.Unlock()
	return nil
}

func (b *outputBase) detach(operation func() int32) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	replay := b.bound
	b.mu.Unlock()
	if replay == nil {
		return ErrNotBound
	}
	replay.op.enter()
	defer replay.op.leave()
	b.op.enter()
	defer b.op.leave()
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	if b.bound == nil || b.bound != replay {
		b.mu.Unlock()
		return ErrNotBound
	}
	b.mu.Unlock()
	if err := nativeError(operation()); err != nil {
		return err
	}
	b.mu.Lock()
	b.bound = nil
	b.dropped = true
	b.mu.Unlock()
	replay.mu.Lock()
	if replay.deps > 0 {
		replay.deps--
	}
	replay.mu.Unlock()
	return nil
}

func (b *outputBase) preclose() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	if b.bound != nil || !b.queue.idle() {
		return ErrInUse
	}
	return nil
}

type AudioOutput struct {
	outputBase
	native  *native.AudioOutput
	options AudioOutputOptions
}

func NewAudioOutput(options AudioOutputOptions) (*AudioOutput, error) {
	if options.OnFrame == nil {
		return nil, ErrInvalidArgument
	}
	o := &AudioOutput{outputBase: newOutputBase(), options: options}
	handle, code := native.NewAudioOutput(0, 0, 0, nil, native.OutputCallbacks{
		OnState: func(state uint32) {
			mapped := outputStateFromNative(state)
			if !o.setState(mapped) {
				return
			}
			logCloudStorageState("cloud_storage_audio_output_state", mapped)
			_ = postOutputState(o.queue, mapped, func() {
				if o.options.OnStateChanged != nil {
					o.options.OnStateChanged(mapped)
				}
			})
		},
		OnError: func(code int32, _ string) {
			logCloudStorageResult("cloud_storage_audio_output", nativeError(code))
			_ = postOutputError(o.queue, func() {
				if o.options.OnError != nil {
					o.options.OnError(nativeError(code))
				}
			})
		},
		OnAudioFrame: func(frame native.AudioFrame) {
			value := AudioFrame{Data: frame.Data, PTS: time.Duration(frame.PTSUs) * time.Microsecond,
				SourceTime: o.sourceTime(frame.SourceTimeUTCUs, frame.HasSourceTime),
				Format:     AudioSampleFormat(frame.Format), SampleRateHz: frame.SampleRateHz,
				Channels: frame.Channels, SamplesPerChannel: frame.SamplesPerChannel,
				Discontinuity: o.discontinuity(frame.Discontinuity)}
			if !o.queue.post(func() { o.options.OnFrame(value) }) {
				o.markDropped()
			}
		},
	})
	if code != 0 {
		o.queue.close()
		return nil, nativeError(code)
	}
	o.native = handle
	return o, nil
}

func (o *AudioOutput) State() OutputState { return o.outputBase.State() }
func (o *AudioOutput) Attach(replay *Replay, channelID uint8) error {
	err := o.attach(replay, channelID, func(r *native.CloudStorageReplay) int32 {
		return o.native.CloudStorageAttach(r, channelID)
	})
	logCloudStorageResult("cloud_storage_audio_attach", err)
	return err
}
func (o *AudioOutput) Detach() error {
	err := o.detach(func() int32 { return o.native.CloudStorageDetach() })
	if err == nil && o.setState(OutputIdle) {
		logCloudStorageState("cloud_storage_audio_output_state", OutputIdle)
		_ = postOutputState(o.queue, OutputIdle, func() {
			if o.options.OnStateChanged != nil {
				o.options.OnStateChanged(OutputIdle)
			}
		})
	}
	logCloudStorageResult("cloud_storage_audio_detach", err)
	return err
}
func (o *AudioOutput) Close() error {
	if !o.queue.idle() {
		return ErrInUse
	}
	o.mu.Lock()
	bound := o.bound != nil
	o.mu.Unlock()
	if bound {
		if err := o.Detach(); err != nil {
			return err
		}
	}
	o.op.enter()
	if err := o.preclose(); err != nil {
		o.op.leave()
		return err
	}
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		o.op.leave()
		return nil
	}
	handle := o.native
	o.mu.Unlock()
	if err := nativeError(handle.Close()); err != nil {
		o.op.leave()
		logCloudStorageResult("cloud_storage_audio_dispose", err)
		return err
	}
	o.mu.Lock()
	o.native = nil
	o.closed = true
	o.mu.Unlock()
	finishNativeClose(&o.op, o.queue)
	logCloudStorageResult("cloud_storage_audio_dispose", nil)
	return nil
}

type VideoOutput struct {
	outputBase
	native  *native.VideoOutput
	options VideoOutputOptions
}

func NewVideoOutput(options VideoOutputOptions) (*VideoOutput, error) {
	o := &VideoOutput{outputBase: newOutputBase(), options: options}
	handle, code := native.NewVideoOutput(0, 0, nil, native.OutputCallbacks{
		OnState: func(state uint32) {
			mapped := outputStateFromNative(state)
			if !o.setState(mapped) {
				return
			}
			logCloudStorageState("cloud_storage_video_output_state", mapped)
			_ = postOutputState(o.queue, mapped, func() {
				if o.options.OnStateChanged != nil {
					o.options.OnStateChanged(mapped)
				}
			})
		},
		OnError: func(code int32, _ string) {
			logCloudStorageResult("cloud_storage_video_output", nativeError(code))
			_ = postOutputError(o.queue, func() {
				if o.options.OnError != nil {
					o.options.OnError(nativeError(code))
				}
			})
		},
		OnVideoFrame: func(frame native.VideoFrame) {
			planes := make([]VideoPlane, 0, len(frame.Planes))
			for _, plane := range frame.Planes {
				planes = append(planes, VideoPlane{Stride: plane.Stride, Data: plane.Data})
			}
			value := VideoFrame{PTS: time.Duration(frame.PTSUs) * time.Microsecond,
				SourceTime:  o.sourceTime(frame.SourceTimeUTCUs, frame.HasSourceTime),
				PixelFormat: PixelFormat(frame.PixelFormat), Width: frame.Width, Height: frame.Height,
				Planes: planes, Discontinuity: o.discontinuity(frame.Discontinuity)}
			if o.options.OnFrame != nil && !o.queue.post(func() { o.options.OnFrame(value) }) {
				o.markDropped()
			}
		},
	})
	if code != 0 {
		o.queue.close()
		return nil, nativeError(code)
	}
	o.native = handle
	return o, nil
}

func (o *VideoOutput) State() OutputState { return o.outputBase.State() }
func (o *VideoOutput) Attach(replay *Replay, channelID uint8) error {
	err := o.attach(replay, channelID, func(r *native.CloudStorageReplay) int32 {
		return o.native.CloudStorageAttach(r, channelID)
	})
	logCloudStorageResult("cloud_storage_video_attach", err)
	return err
}
func (o *VideoOutput) Detach() error {
	err := o.detach(func() int32 { return o.native.CloudStorageDetach() })
	if err == nil && o.setState(OutputIdle) {
		logCloudStorageState("cloud_storage_video_output_state", OutputIdle)
		_ = postOutputState(o.queue, OutputIdle, func() {
			if o.options.OnStateChanged != nil {
				o.options.OnStateChanged(OutputIdle)
			}
		})
	}
	logCloudStorageResult("cloud_storage_video_detach", err)
	return err
}
func (o *VideoOutput) TakeSnapshot() (SnapshotFile, error) {
	o.op.enter()
	defer o.op.leave()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return SnapshotFile{}, ErrClosed
	}
	handle := o.native
	o.mu.Unlock()
	path, code := handle.Snapshot()
	err := nativeError(code)
	logCloudStorageResult("cloud_storage_snapshot", err)
	if err != nil {
		return SnapshotFile{}, err
	}
	return SnapshotFile{Path: path}, nil
}
func (o *VideoOutput) Close() error {
	if !o.queue.idle() {
		return ErrInUse
	}
	o.mu.Lock()
	bound := o.bound != nil
	o.mu.Unlock()
	if bound {
		if err := o.Detach(); err != nil {
			return err
		}
	}
	o.op.enter()
	if err := o.preclose(); err != nil {
		o.op.leave()
		return err
	}
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		o.op.leave()
		return nil
	}
	handle := o.native
	o.mu.Unlock()
	if err := nativeError(handle.Close()); err != nil {
		o.op.leave()
		logCloudStorageResult("cloud_storage_video_dispose", err)
		return err
	}
	o.mu.Lock()
	o.native = nil
	o.closed = true
	o.mu.Unlock()
	finishNativeClose(&o.op, o.queue)
	logCloudStorageResult("cloud_storage_video_dispose", nil)
	return nil
}

type EncodedAudioOutput struct {
	outputBase
	native  *native.EncodedAudioOutput
	options EncodedAudioOutputOptions
}

func NewEncodedAudioOutput(options EncodedAudioOutputOptions) (*EncodedAudioOutput, error) {
	if options.OnFrame == nil {
		return nil, ErrInvalidArgument
	}
	o := &EncodedAudioOutput{outputBase: newOutputBase(), options: options}
	handle, code := native.NewEncodedAudioOutput(native.OutputCallbacks{
		OnState: func(state uint32) {
			mapped := outputStateFromNative(state)
			if !o.setState(mapped) {
				return
			}
			logCloudStorageState("cloud_storage_encoded_audio_output_state", mapped)
			_ = postOutputState(o.queue, mapped, func() {
				if o.options.OnStateChanged != nil {
					o.options.OnStateChanged(mapped)
				}
			})
		},
		OnError: func(code int32, _ string) {
			logCloudStorageResult("cloud_storage_encoded_audio_output", nativeError(code))
			_ = postOutputError(o.queue, func() {
				if o.options.OnError != nil {
					o.options.OnError(nativeError(code))
				}
			})
		},
		OnEncodedAudio: func(frame native.EncodedAudioFrame) {
			value := EncodedAudioFrame{Data: frame.Data, CodecConfig: frame.CodecConfig,
				PTS:        time.Duration(frame.PTSUs) * time.Microsecond,
				SourceTime: o.sourceTime(frame.SourceTimeUTCUs, frame.HasSourceTime), Codec: AudioCodec(frame.Codec),
				BitstreamFormat: AudioBitstreamFormat(frame.Bitstream), SampleRateHz: frame.SampleRateHz,
				Channels: frame.Channels, Discontinuity: o.discontinuity(frame.Discontinuity)}
			if !o.queue.post(func() { o.options.OnFrame(value) }) {
				o.markDropped()
			}
		},
	})
	if code != 0 {
		o.queue.close()
		return nil, nativeError(code)
	}
	o.native = handle
	return o, nil
}

func (o *EncodedAudioOutput) State() OutputState { return o.outputBase.State() }
func (o *EncodedAudioOutput) Attach(replay *Replay, channelID uint8) error {
	err := o.attach(replay, channelID, func(r *native.CloudStorageReplay) int32 {
		return o.native.CloudStorageAttach(r, channelID)
	})
	logCloudStorageResult("cloud_storage_encoded_audio_attach", err)
	return err
}
func (o *EncodedAudioOutput) Detach() error {
	err := o.detach(func() int32 { return o.native.CloudStorageDetach() })
	if err == nil && o.setState(OutputIdle) {
		logCloudStorageState("cloud_storage_encoded_audio_output_state", OutputIdle)
		_ = postOutputState(o.queue, OutputIdle, func() {
			if o.options.OnStateChanged != nil {
				o.options.OnStateChanged(OutputIdle)
			}
		})
	}
	logCloudStorageResult("cloud_storage_encoded_audio_detach", err)
	return err
}
func (o *EncodedAudioOutput) Close() error {
	if !o.queue.idle() {
		return ErrInUse
	}
	o.mu.Lock()
	bound := o.bound != nil
	o.mu.Unlock()
	if bound {
		if err := o.Detach(); err != nil {
			return err
		}
	}
	o.op.enter()
	if err := o.preclose(); err != nil {
		o.op.leave()
		return err
	}
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		o.op.leave()
		return nil
	}
	handle := o.native
	o.mu.Unlock()
	if err := nativeError(handle.Close()); err != nil {
		o.op.leave()
		logCloudStorageResult("cloud_storage_encoded_audio_dispose", err)
		return err
	}
	o.mu.Lock()
	o.native = nil
	o.closed = true
	o.mu.Unlock()
	finishNativeClose(&o.op, o.queue)
	logCloudStorageResult("cloud_storage_encoded_audio_dispose", nil)
	return nil
}

type EncodedVideoOutput struct {
	outputBase
	native  *native.EncodedVideoOutput
	options EncodedVideoOutputOptions
}

func NewEncodedVideoOutput(options EncodedVideoOutputOptions) (*EncodedVideoOutput, error) {
	if options.OnFrame == nil {
		return nil, ErrInvalidArgument
	}
	o := &EncodedVideoOutput{outputBase: newOutputBase(), options: options}
	handle, code := native.NewEncodedVideoOutput(native.OutputCallbacks{
		OnState: func(state uint32) {
			mapped := outputStateFromNative(state)
			if !o.setState(mapped) {
				return
			}
			logCloudStorageState("cloud_storage_encoded_video_output_state", mapped)
			_ = postOutputState(o.queue, mapped, func() {
				if o.options.OnStateChanged != nil {
					o.options.OnStateChanged(mapped)
				}
			})
		},
		OnError: func(code int32, _ string) {
			logCloudStorageResult("cloud_storage_encoded_video_output", nativeError(code))
			_ = postOutputError(o.queue, func() {
				if o.options.OnError != nil {
					o.options.OnError(nativeError(code))
				}
			})
		},
		OnEncodedVideo: func(frame native.EncodedVideoFrame) {
			value := EncodedVideoFrame{Data: frame.Data, CodecConfig: frame.CodecConfig, PTS: time.Duration(frame.PTSUs) * time.Microsecond,
				SourceTime: o.sourceTime(frame.SourceTimeUTCUs, frame.HasSourceTime), Codec: VideoCodec(frame.Codec),
				BitstreamFormat: VideoBitstreamFormat(frame.Bitstream), Width: frame.Width, Height: frame.Height,
				KeyFrame: frame.KeyFrame, Discontinuity: o.discontinuity(frame.Discontinuity)}
			if !o.queue.post(func() { o.options.OnFrame(value) }) {
				o.markDropped()
			}
		},
	})
	if code != 0 {
		o.queue.close()
		return nil, nativeError(code)
	}
	o.native = handle
	return o, nil
}

func (o *EncodedVideoOutput) State() OutputState { return o.outputBase.State() }
func (o *EncodedVideoOutput) Attach(replay *Replay, channelID uint8) error {
	err := o.attach(replay, channelID, func(r *native.CloudStorageReplay) int32 {
		return o.native.CloudStorageAttach(r, channelID)
	})
	logCloudStorageResult("cloud_storage_encoded_video_attach", err)
	return err
}
func (o *EncodedVideoOutput) Detach() error {
	err := o.detach(func() int32 { return o.native.CloudStorageDetach() })
	if err == nil && o.setState(OutputIdle) {
		logCloudStorageState("cloud_storage_encoded_video_output_state", OutputIdle)
		_ = postOutputState(o.queue, OutputIdle, func() {
			if o.options.OnStateChanged != nil {
				o.options.OnStateChanged(OutputIdle)
			}
		})
	}
	logCloudStorageResult("cloud_storage_encoded_video_detach", err)
	return err
}
func (o *EncodedVideoOutput) Close() error {
	if !o.queue.idle() {
		return ErrInUse
	}
	o.mu.Lock()
	bound := o.bound != nil
	o.mu.Unlock()
	if bound {
		if err := o.Detach(); err != nil {
			return err
		}
	}
	o.op.enter()
	if err := o.preclose(); err != nil {
		o.op.leave()
		return err
	}
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		o.op.leave()
		return nil
	}
	handle := o.native
	o.mu.Unlock()
	if err := nativeError(handle.Close()); err != nil {
		o.op.leave()
		logCloudStorageResult("cloud_storage_encoded_video_dispose", err)
		return err
	}
	o.mu.Lock()
	o.native = nil
	o.closed = true
	o.mu.Unlock()
	finishNativeClose(&o.op, o.queue)
	logCloudStorageResult("cloud_storage_encoded_video_dispose", nil)
	return nil
}
