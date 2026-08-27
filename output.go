package tirtc

import (
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type AudioOutputOptions struct {
	AGCLevel       AudioProcessingLevel
	ANSLevel       AudioProcessingLevel
	Buffer         OutputBufferOptions
	OnFrame        func(frame AudioFrame)
	OnStateChanged func(state OutputState)
	OnError        func(err error)
}

type VideoOutputOptions struct {
	DecoderPreference VideoDecoderPreference
	Buffer            OutputBufferOptions
	OnFrame           func(frame VideoFrame)
	OnStateChanged    func(state OutputState)
	OnError           func(err error)
}

type EncodedAudioOutputOptions struct {
	OnFrame        func(frame EncodedAudioFrame)
	OnStateChanged func(state OutputState)
	OnError        func(err error)
}

type EncodedVideoOutputOptions struct {
	OnFrame        func(frame EncodedVideoFrame)
	OnStateChanged func(state OutputState)
	OnError        func(err error)
}

type outputBase struct {
	opMu     sync.Mutex
	mu       sync.Mutex
	mailbox  mailbox
	state    OutputState
	closed   bool
	bound    *Conn
	dropped  bool
	snapshot int
}

func sourceTime(value int64, present bool) time.Time {
	if !present {
		return time.Time{}
	}
	return time.UnixMicro(value).UTC()
}

func (b *outputBase) stateValue() OutputState { b.mu.Lock(); defer b.mu.Unlock(); return b.state }
func (b *outputBase) setState(state OutputState) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == state {
		return false
	}
	b.state = state
	return true
}
func (b *outputBase) markDropped() { b.mu.Lock(); b.dropped = true; b.mu.Unlock() }
func (b *outputBase) discontinuity(value bool) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	value = value || b.dropped
	b.dropped = false
	return value
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
	output := &AudioOutput{options: options}
	handle, code := native.NewAudioOutput(uint32(options.AGCLevel), uint32(options.ANSLevel), uint32(options.Buffer.Strategy), options.Buffer.MaxBufferWatermark, native.OutputCallbacks{
		OnState: func(state uint32) {
			if !output.setState(OutputState(state)) {
				return
			}
			logSDKState("audio_output_state", state, nil)
			if output.options.OnStateChanged != nil {
				_ = output.mailbox.postLatest(mailboxLatestState, func() {
					output.options.OnStateChanged(OutputState(state))
				})
			}
		},
		OnError: func(code int32, _ string) {
			logSDKResult("audio_output", nativeError(code))
			if output.options.OnError != nil {
				if !output.mailbox.postControl(func() {
					output.options.OnError(nativeError(code))
				}) {
					_ = output.mailbox.postStopRequest(func() { _ = output.Detach() })
				}
			}
		},
		OnAudioFrame: func(frame native.AudioFrame) {
			value := AudioFrame{Data: frame.Data, PTS: time.Duration(frame.PTSUs) * time.Microsecond, SourceTime: sourceTime(frame.SourceTimeUTCUs, frame.HasSourceTime), Format: AudioSampleFormat(frame.Format), SampleRateHz: frame.SampleRateHz, Channels: frame.Channels, SamplesPerChannel: frame.SamplesPerChannel, Discontinuity: output.discontinuity(frame.Discontinuity)}
			if !output.mailbox.postData(func() { output.options.OnFrame(value) }) {
				output.markDropped()
			}
		},
	})
	if code != 0 {
		err := nativeError(code)
		logSDKResult("audio_output_create", err)
		return nil, err
	}
	output.native = handle
	logSDKResult("audio_output_create", nil)
	return output, nil
}

func (o *AudioOutput) State() OutputState { return o.stateValue() }
func (o *AudioOutput) Attach(connection *Conn, streamID uint8) (resultErr error) {
	defer func() { logSDKResult("audio_output_attach", resultErr) }()
	if connection == nil || !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	o.opMu.Lock()
	defer o.opMu.Unlock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ErrClosed
	}
	if o.bound != nil {
		o.mu.Unlock()
		return ErrInUse
	}
	handle := o.native
	o.mu.Unlock()
	if err := connection.attachDependency(func(connHandle *native.Conn) error {
		return nativeError(handle.Attach(connHandle, streamID))
	}); err != nil {
		return err
	}
	o.mu.Lock()
	o.bound = connection
	o.mu.Unlock()
	return nil
}
func (o *AudioOutput) Detach() (resultErr error) {
	defer func() { logSDKResult("audio_output_detach", resultErr) }()
	o.opMu.Lock()
	defer o.opMu.Unlock()
	return o.detachLocked()
}

func (o *AudioOutput) detachLocked() error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ErrClosed
	}
	connection := o.bound
	if connection == nil {
		o.mu.Unlock()
		return ErrNotBound
	}
	handle := o.native
	o.mu.Unlock()
	if err := connection.detachDependency(func(*native.Conn) error {
		return nativeError(handle.Detach())
	}); err != nil {
		return err
	}
	o.mu.Lock()
	o.bound = nil
	o.mu.Unlock()
	if o.setState(OutputIdle) {
		logSDKState("audio_output_state", uint32(OutputIdle), nil)
		if o.options.OnStateChanged != nil {
			_ = o.mailbox.postLatest(mailboxLatestState, func() {
				o.options.OnStateChanged(OutputIdle)
			})
		}
	}
	return nil
}
func (o *AudioOutput) Close() (resultErr error) {
	defer func() { logSDKResult("audio_output_dispose", resultErr) }()
	o.mu.Lock()
	alreadyClosed := o.closed
	o.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	if err := o.mailbox.preflightClose(); err != nil {
		return err
	}
	o.opMu.Lock()
	defer o.opMu.Unlock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return nil
	}
	if err := o.mailbox.beginClose(); err != nil {
		o.mu.Unlock()
		return err
	}
	bound := o.bound
	handle := o.native
	o.mu.Unlock()
	if bound != nil {
		if err := o.detachLocked(); err != nil {
			o.mailbox.cancelClose()
			return err
		}
	}
	if err := nativeError(handle.Close()); err != nil {
		o.mailbox.cancelClose()
		return err
	}
	o.mu.Lock()
	o.closed = true
	o.native = nil
	o.mu.Unlock()
	o.mailbox.finishClose()
	return nil
}

type VideoOutput struct {
	outputBase
	native  *native.VideoOutput
	options VideoOutputOptions
}

func NewVideoOutput(options VideoOutputOptions) (*VideoOutput, error) {
	output := &VideoOutput{options: options}
	handle, code := native.NewVideoOutput(uint32(options.DecoderPreference), uint32(options.Buffer.Strategy), options.Buffer.MaxBufferWatermark, native.OutputCallbacks{
		OnState: func(state uint32) {
			if !output.setState(OutputState(state)) {
				return
			}
			logSDKState("video_output_state", state, nil)
			if output.options.OnStateChanged != nil {
				_ = output.mailbox.postLatest(mailboxLatestState, func() {
					output.options.OnStateChanged(OutputState(state))
				})
			}
		},
		OnError: func(code int32, _ string) {
			logSDKResult("video_output", nativeError(code))
			if output.options.OnError != nil {
				if !output.mailbox.postControl(func() {
					output.options.OnError(nativeError(code))
				}) {
					_ = output.mailbox.postStopRequest(func() { _ = output.Detach() })
				}
			}
		},
		OnVideoFrame: func(frame native.VideoFrame) {
			planes := make([]VideoPlane, len(frame.Planes))
			for index, plane := range frame.Planes {
				planes[index] = VideoPlane{Stride: plane.Stride, Data: plane.Data}
			}
			value := VideoFrame{PTS: time.Duration(frame.PTSUs) * time.Microsecond, SourceTime: sourceTime(frame.SourceTimeUTCUs, frame.HasSourceTime), PixelFormat: PixelFormat(frame.PixelFormat), Width: frame.Width, Height: frame.Height, Planes: planes, Discontinuity: output.discontinuity(frame.Discontinuity)}
			if output.options.OnFrame != nil && !output.mailbox.postData(func() { output.options.OnFrame(value) }) {
				output.markDropped()
			}
		},
	})
	if code != 0 {
		err := nativeError(code)
		logSDKResult("video_output_create", err)
		return nil, err
	}
	output.native = handle
	logSDKResult("video_output_create", nil)
	return output, nil
}
func (o *VideoOutput) State() OutputState { return o.stateValue() }
func (o *VideoOutput) Attach(connection *Conn, streamID uint8) (resultErr error) {
	defer func() { logSDKResult("video_output_attach", resultErr) }()
	if connection == nil || !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	o.opMu.Lock()
	defer o.opMu.Unlock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ErrClosed
	}
	if o.bound != nil {
		o.mu.Unlock()
		return ErrInUse
	}
	if o.snapshot != 0 {
		o.mu.Unlock()
		return ErrInUse
	}
	handle := o.native
	o.mu.Unlock()
	if err := connection.attachDependency(func(connHandle *native.Conn) error {
		return nativeError(handle.Attach(connHandle, streamID))
	}); err != nil {
		return err
	}
	o.mu.Lock()
	o.bound = connection
	o.mu.Unlock()
	return nil
}
func (o *VideoOutput) Detach() (resultErr error) {
	defer func() { logSDKResult("video_output_detach", resultErr) }()
	o.opMu.Lock()
	defer o.opMu.Unlock()
	return o.detachLocked()
}

func (o *VideoOutput) detachLocked() error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ErrClosed
	}
	if o.snapshot != 0 {
		o.mu.Unlock()
		return ErrInUse
	}
	connection := o.bound
	if connection == nil {
		o.mu.Unlock()
		return ErrNotBound
	}
	handle := o.native
	o.mu.Unlock()
	if err := connection.detachDependency(func(*native.Conn) error {
		return nativeError(handle.Detach())
	}); err != nil {
		return err
	}
	o.mu.Lock()
	o.bound = nil
	o.mu.Unlock()
	if o.setState(OutputIdle) {
		logSDKState("video_output_state", uint32(OutputIdle), nil)
		if o.options.OnStateChanged != nil {
			_ = o.mailbox.postLatest(mailboxLatestState, func() {
				o.options.OnStateChanged(OutputIdle)
			})
		}
	}
	return nil
}
func (o *VideoOutput) TakeSnapshot() (file SnapshotFile, resultErr error) {
	defer func() { logSDKResult("video_output_snapshot", resultErr) }()
	o.opMu.Lock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		o.opMu.Unlock()
		return SnapshotFile{}, ErrClosed
	}
	if o.snapshot != 0 {
		o.mu.Unlock()
		o.opMu.Unlock()
		return SnapshotFile{}, ErrInUse
	}
	o.snapshot++
	handle := o.native
	o.mu.Unlock()
	o.opMu.Unlock()
	path, code := handle.Snapshot()
	o.opMu.Lock()
	o.mu.Lock()
	o.snapshot--
	o.mu.Unlock()
	o.opMu.Unlock()
	if code != 0 {
		return SnapshotFile{}, nativeError(code)
	}
	return SnapshotFile{Path: path}, nil
}
func (o *VideoOutput) Close() (resultErr error) {
	defer func() { logSDKResult("video_output_dispose", resultErr) }()
	o.mu.Lock()
	alreadyClosed := o.closed
	snapshotInFlight := o.snapshot != 0
	o.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	if snapshotInFlight {
		return ErrInUse
	}
	if err := o.mailbox.preflightClose(); err != nil {
		return err
	}
	o.opMu.Lock()
	defer o.opMu.Unlock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return nil
	}
	if o.snapshot != 0 {
		o.mu.Unlock()
		return ErrInUse
	}
	if err := o.mailbox.beginClose(); err != nil {
		o.mu.Unlock()
		return err
	}
	bound := o.bound
	handle := o.native
	o.mu.Unlock()
	if bound != nil {
		if err := o.detachLocked(); err != nil {
			o.mailbox.cancelClose()
			return err
		}
	}
	if err := nativeError(handle.Close()); err != nil {
		o.mailbox.cancelClose()
		return err
	}
	o.mu.Lock()
	o.closed = true
	o.native = nil
	o.mu.Unlock()
	o.mailbox.finishClose()
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
	output := &EncodedAudioOutput{options: options}
	handle, code := native.NewEncodedAudioOutput(native.OutputCallbacks{OnState: func(state uint32) {
		if !output.setState(OutputState(state)) {
			return
		}
		logSDKState("encoded_audio_output_state", state, nil)
		if output.options.OnStateChanged != nil {
			_ = output.mailbox.postLatest(mailboxLatestState, func() {
				output.options.OnStateChanged(OutputState(state))
			})
		}
	}, OnError: func(code int32, _ string) {
		logSDKResult("encoded_audio_output", nativeError(code))
		if output.options.OnError != nil {
			if !output.mailbox.postControl(func() {
				output.options.OnError(nativeError(code))
			}) {
				_ = output.mailbox.postStopRequest(func() { _ = output.Detach() })
			}
		}
	}, OnEncodedAudio: func(frame native.EncodedAudioFrame) {
		value := EncodedAudioFrame{Data: frame.Data, CodecConfig: frame.CodecConfig, PTS: time.Duration(frame.PTSUs) * time.Microsecond, SourceTime: sourceTime(frame.SourceTimeUTCUs, frame.HasSourceTime), Codec: AudioCodec(frame.Codec), BitstreamFormat: AudioBitstreamFormat(frame.Bitstream), SampleRateHz: frame.SampleRateHz, Channels: frame.Channels, Discontinuity: output.discontinuity(frame.Discontinuity)}
		if !output.mailbox.postData(func() { output.options.OnFrame(value) }) {
			output.markDropped()
		}
	}})
	if code != 0 {
		err := nativeError(code)
		logSDKResult("encoded_audio_output_create", err)
		return nil, err
	}
	output.native = handle
	logSDKResult("encoded_audio_output_create", nil)
	return output, nil
}
func (o *EncodedAudioOutput) State() OutputState { return o.stateValue() }
func (o *EncodedAudioOutput) Attach(connection *Conn, streamID uint8) (resultErr error) {
	defer func() { logSDKResult("encoded_audio_output_attach", resultErr) }()
	if connection == nil || !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	o.opMu.Lock()
	defer o.opMu.Unlock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ErrClosed
	}
	if o.bound != nil {
		o.mu.Unlock()
		return ErrInUse
	}
	handle := o.native
	o.mu.Unlock()
	if err := connection.attachDependency(func(connHandle *native.Conn) error {
		return nativeError(handle.Attach(connHandle, streamID))
	}); err != nil {
		return err
	}
	o.mu.Lock()
	o.bound = connection
	o.mu.Unlock()
	return nil
}
func (o *EncodedAudioOutput) Detach() (resultErr error) {
	defer func() { logSDKResult("encoded_audio_output_detach", resultErr) }()
	o.opMu.Lock()
	defer o.opMu.Unlock()
	return o.detachLocked()
}

func (o *EncodedAudioOutput) detachLocked() error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ErrClosed
	}
	connection := o.bound
	if connection == nil {
		o.mu.Unlock()
		return ErrNotBound
	}
	handle := o.native
	o.mu.Unlock()
	if err := connection.detachDependency(func(*native.Conn) error {
		return nativeError(handle.Detach())
	}); err != nil {
		return err
	}
	o.mu.Lock()
	o.bound = nil
	o.mu.Unlock()
	if o.setState(OutputIdle) {
		logSDKState("encoded_audio_output_state", uint32(OutputIdle), nil)
		if o.options.OnStateChanged != nil {
			_ = o.mailbox.postLatest(mailboxLatestState, func() {
				o.options.OnStateChanged(OutputIdle)
			})
		}
	}
	return nil
}
func (o *EncodedAudioOutput) Close() (resultErr error) {
	defer func() { logSDKResult("encoded_audio_output_dispose", resultErr) }()
	o.mu.Lock()
	alreadyClosed := o.closed
	o.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	if err := o.mailbox.preflightClose(); err != nil {
		return err
	}
	o.opMu.Lock()
	defer o.opMu.Unlock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return nil
	}
	if err := o.mailbox.beginClose(); err != nil {
		o.mu.Unlock()
		return err
	}
	bound := o.bound
	handle := o.native
	o.mu.Unlock()
	if bound != nil {
		if err := o.detachLocked(); err != nil {
			o.mailbox.cancelClose()
			return err
		}
	}
	if err := nativeError(handle.Close()); err != nil {
		o.mailbox.cancelClose()
		return err
	}
	o.mu.Lock()
	o.closed = true
	o.native = nil
	o.mu.Unlock()
	o.mailbox.finishClose()
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
	output := &EncodedVideoOutput{options: options}
	handle, code := native.NewEncodedVideoOutput(native.OutputCallbacks{OnState: func(state uint32) {
		if !output.setState(OutputState(state)) {
			return
		}
		logSDKState("encoded_video_output_state", state, nil)
		if output.options.OnStateChanged != nil {
			_ = output.mailbox.postLatest(mailboxLatestState, func() {
				output.options.OnStateChanged(OutputState(state))
			})
		}
	}, OnError: func(code int32, _ string) {
		logSDKResult("encoded_video_output", nativeError(code))
		if output.options.OnError != nil {
			if !output.mailbox.postControl(func() {
				output.options.OnError(nativeError(code))
			}) {
				_ = output.mailbox.postStopRequest(func() { _ = output.Detach() })
			}
		}
	}, OnEncodedVideo: func(frame native.EncodedVideoFrame) {
		value := EncodedVideoFrame{Data: frame.Data, CodecConfig: frame.CodecConfig, PTS: time.Duration(frame.PTSUs) * time.Microsecond, SourceTime: sourceTime(frame.SourceTimeUTCUs, frame.HasSourceTime), Codec: VideoCodec(frame.Codec), BitstreamFormat: VideoBitstreamFormat(frame.Bitstream), Width: frame.Width, Height: frame.Height, KeyFrame: frame.KeyFrame, Discontinuity: output.discontinuity(frame.Discontinuity)}
		if !output.mailbox.postData(func() { output.options.OnFrame(value) }) {
			output.markDropped()
		}
	}})
	if code != 0 {
		err := nativeError(code)
		logSDKResult("encoded_video_output_create", err)
		return nil, err
	}
	output.native = handle
	logSDKResult("encoded_video_output_create", nil)
	return output, nil
}
func (o *EncodedVideoOutput) State() OutputState { return o.stateValue() }
func (o *EncodedVideoOutput) Attach(connection *Conn, streamID uint8) (resultErr error) {
	defer func() { logSDKResult("encoded_video_output_attach", resultErr) }()
	if connection == nil || !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	o.opMu.Lock()
	defer o.opMu.Unlock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ErrClosed
	}
	if o.bound != nil {
		o.mu.Unlock()
		return ErrInUse
	}
	handle := o.native
	o.mu.Unlock()
	if err := connection.attachDependency(func(connHandle *native.Conn) error {
		return nativeError(handle.Attach(connHandle, streamID))
	}); err != nil {
		return err
	}
	o.mu.Lock()
	o.bound = connection
	o.mu.Unlock()
	return nil
}
func (o *EncodedVideoOutput) Detach() (resultErr error) {
	defer func() { logSDKResult("encoded_video_output_detach", resultErr) }()
	o.opMu.Lock()
	defer o.opMu.Unlock()
	return o.detachLocked()
}

func (o *EncodedVideoOutput) detachLocked() error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return ErrClosed
	}
	connection := o.bound
	if connection == nil {
		o.mu.Unlock()
		return ErrNotBound
	}
	handle := o.native
	o.mu.Unlock()
	if err := connection.detachDependency(func(*native.Conn) error {
		return nativeError(handle.Detach())
	}); err != nil {
		return err
	}
	o.mu.Lock()
	o.bound = nil
	o.mu.Unlock()
	if o.setState(OutputIdle) {
		logSDKState("encoded_video_output_state", uint32(OutputIdle), nil)
		if o.options.OnStateChanged != nil {
			_ = o.mailbox.postLatest(mailboxLatestState, func() {
				o.options.OnStateChanged(OutputIdle)
			})
		}
	}
	return nil
}
func (o *EncodedVideoOutput) Close() (resultErr error) {
	defer func() { logSDKResult("encoded_video_output_dispose", resultErr) }()
	o.mu.Lock()
	alreadyClosed := o.closed
	o.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	if err := o.mailbox.preflightClose(); err != nil {
		return err
	}
	o.opMu.Lock()
	defer o.opMu.Unlock()
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return nil
	}
	if err := o.mailbox.beginClose(); err != nil {
		o.mu.Unlock()
		return err
	}
	bound := o.bound
	handle := o.native
	o.mu.Unlock()
	if bound != nil {
		if err := o.detachLocked(); err != nil {
			o.mailbox.cancelClose()
			return err
		}
	}
	if err := nativeError(handle.Close()); err != nil {
		o.mailbox.cancelClose()
		return err
	}
	o.mu.Lock()
	o.closed = true
	o.native = nil
	o.mu.Unlock()
	o.mailbox.finishClose()
	return nil
}
