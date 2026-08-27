package native

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"runtime/cgo"
	"sync"
	"time"
	"unsafe"
)

type InitOptions struct {
	AppID, Endpoint, CacheDir string
	ConsoleLogEnabled         bool
}

func cString(value string) (*C.char, func()) {
	if value == "" {
		return nil, func() {}
	}
	result := C.CString(value)
	return result, func() { C.free(unsafe.Pointer(result)) }
}

func Init(options InitOptions) int32 {
	appID, freeAppID := cString(options.AppID)
	defer freeAppID()
	endpoint, freeEndpoint := cString(options.Endpoint)
	defer freeEndpoint()
	cache, freeCache := cString(options.CacheDir)
	defer freeCache()
	return int32(C.ti_go_init(appID, endpoint, cache, boolByte(options.ConsoleLogEnabled)))
}

func Shutdown() int32 { return int32(C.tirtc_uninit()) }

func ErrorName(code int32) string { return C.GoString(C.ti_error_to_string(C.TiError(code))) }

func Log(level uint32, tag, message string) int32 {
	cTag, freeTag := cString(tag)
	defer freeTag()
	cMessage, freeMessage := cString(message)
	defer freeMessage()
	return int32(C.ti_logging_write(C.TiLogLevel(level), cTag, cMessage))
}

func UploadLogs() (string, int32) {
	buffer := make([]byte, 256)
	code := int32(C.ti_go_logging_upload((*C.char)(unsafe.Pointer(&buffer[0])), C.uint32_t(len(buffer))))
	if code != 0 {
		return "", code
	}
	return C.GoString((*C.char)(unsafe.Pointer(&buffer[0]))), 0
}

func DeleteMediaFile(path string) int32 {
	value, freeValue := cString(path)
	defer freeValue()
	return int32(C.ti_local_media_file_delete(value))
}

type ConnCallbacks struct {
	OnState   func(uint32, int32)
	OnCommand func(uint32, []byte)
	OnMessage func(uint8, uint32, []byte)
}

type connContext struct {
	mu        sync.Mutex
	callbacks *ConnCallbacks
	pending   []func(ConnCallbacks)
}

func (c *connContext) emit(event func(ConnCallbacks)) {
	c.mu.Lock()
	if c.callbacks == nil {
		if len(c.pending) < 256 {
			c.pending = append(c.pending, event)
		}
		c.mu.Unlock()
		return
	}
	callbacks := *c.callbacks
	c.mu.Unlock()
	event(callbacks)
}

func (c *connContext) setCallbacks(callbacks ConnCallbacks) {
	c.mu.Lock()
	c.callbacks = &callbacks
	pending := c.pending
	c.pending = nil
	c.mu.Unlock()
	for _, event := range pending {
		event(callbacks)
	}
}

type Conn struct {
	ptr    *C.TiRtcConn
	handle cgo.Handle
}

func NewConn(callbacks ConnCallbacks) (*Conn, int32) {
	context := &connContext{callbacks: &callbacks}
	handle := cgo.NewHandle(context)
	var pointer *C.TiRtcConn
	code := int32(C.ti_go_conn_create(C.uintptr_t(handle), &pointer))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &Conn{ptr: pointer, handle: handle}, 0
}

func (c *Conn) Connect(remoteID, token string) int32 {
	remote, freeRemote := cString(remoteID)
	defer freeRemote()
	cToken, freeToken := cString(token)
	defer freeToken()
	options := C.TiRtcConnConnectOptions{
		remote_id: remote,
		token:     cToken,
	}
	return int32(C.tirtc_conn_connect(c.ptr, &options))
}

func (c *Conn) Disconnect() int32 { return int32(C.tirtc_conn_disconnect(c.ptr)) }

func bytesPointer(data []byte) *C.uint8_t {
	if len(data) == 0 {
		return nil
	}
	return (*C.uint8_t)(unsafe.Pointer(&data[0]))
}

func (c *Conn) SendCommand(commandID uint32, data []byte) int32 {
	return int32(C.ti_go_conn_send_command(c.ptr, C.uint32_t(commandID), bytesPointer(data), C.uint64_t(len(data))))
}

func (c *Conn) SendMessage(streamID uint8, timestampMs uint32, data []byte) int32 {
	return int32(C.ti_go_conn_send_message(c.ptr, C.uint8_t(streamID), C.uint32_t(timestampMs), bytesPointer(data), C.uint64_t(len(data))))
}

func (c *Conn) SubscribeAudio(id uint8) int32 {
	return int32(C.tirtc_conn_subscribe_audio(c.ptr, C.uint8_t(id)))
}
func (c *Conn) UnsubscribeAudio(id uint8) int32 {
	return int32(C.tirtc_conn_unsubscribe_audio(c.ptr, C.uint8_t(id)))
}
func (c *Conn) SubscribeVideo(id uint8) int32 {
	return int32(C.tirtc_conn_subscribe_video(c.ptr, C.uint8_t(id)))
}
func (c *Conn) UnsubscribeVideo(id uint8) int32 {
	return int32(C.tirtc_conn_unsubscribe_video(c.ptr, C.uint8_t(id)))
}
func (c *Conn) RequestVideoKeyframe(id uint8) int32 {
	return int32(C.tirtc_conn_request_video_key_frame(c.ptr, C.uint8_t(id)))
}

func (c *Conn) Close() int32 {
	code := int32(C.tirtc_conn_destroy(c.ptr))
	if code == 0 {
		c.ptr = nil
		c.handle.Delete()
	}
	return code
}

type AudioFrame struct {
	Data                                      []byte
	PTSUs                                     int64
	SourceTimeUTCUs                           int64
	HasSourceTime                             bool
	Format                                    uint32
	SampleRateHz, Channels, SamplesPerChannel int
	Discontinuity                             bool
}
type VideoPlane struct {
	Stride int
	Data   []byte
}
type VideoFrame struct {
	PTSUs, SourceTimeUTCUs int64
	HasSourceTime          bool
	PixelFormat            uint32
	Width, Height          int
	Planes                 []VideoPlane
	Discontinuity          bool
}
type EncodedAudioFrame struct {
	Data, CodecConfig      []byte
	PTSUs, SourceTimeUTCUs int64
	HasSourceTime          bool
	Codec, Bitstream       uint32
	SampleRateHz, Channels int
	Discontinuity          bool
}
type EncodedVideoFrame struct {
	Data, CodecConfig       []byte
	PTSUs, SourceTimeUTCUs  int64
	HasSourceTime           bool
	Codec, Bitstream        uint32
	Width, Height           int
	KeyFrame, Discontinuity bool
}

type OutputCallbacks struct {
	OnState        func(uint32)
	OnError        func(int32, string)
	OnAudioFrame   func(AudioFrame)
	OnVideoFrame   func(VideoFrame)
	OnEncodedAudio func(EncodedAudioFrame)
	OnEncodedVideo func(EncodedVideoFrame)
}
type outputContext struct{ callbacks OutputCallbacks }

type AudioOutput struct {
	ptr    *C.TiAudioOutput
	handle cgo.Handle
}
type VideoOutput struct {
	ptr    *C.TiVideoOutput
	handle cgo.Handle
}
type EncodedAudioOutput struct {
	ptr    *C.TiEncodedAudioOutput
	handle cgo.Handle
}
type EncodedVideoOutput struct {
	ptr    *C.TiEncodedVideoOutput
	handle cgo.Handle
}

func buffer(optionsStrategy uint32, watermark *time.Duration) (C.uint8_t, C.int32_t) {
	if watermark == nil {
		return 0, 0
	}
	return 1, C.int32_t(watermark.Milliseconds())
}

func NewAudioOutput(agc, ans, strategy uint32, watermark *time.Duration, callbacks OutputCallbacks) (*AudioOutput, int32) {
	has, value := buffer(strategy, watermark)
	context := &outputContext{callbacks: callbacks}
	handle := cgo.NewHandle(context)
	var pointer *C.TiAudioOutput
	code := int32(C.ti_go_audio_output_create(C.TiAudioAgcLevel(agc), C.TiAudioAnsLevel(ans), C.TiOutputBufferStrategy(strategy), has, value, C.uintptr_t(handle), &pointer))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &AudioOutput{ptr: pointer, handle: handle}, 0
}
func (o *AudioOutput) Attach(c *Conn, id uint8) int32 {
	return int32(C.tirtc_audio_output_attach(o.ptr, c.ptr, C.uint8_t(id)))
}
func (o *AudioOutput) Detach() int32 { return int32(C.tirtc_audio_output_detach(o.ptr)) }
func (o *AudioOutput) State() (uint32, int32) {
	var state C.TiOutputState
	code := int32(C.ti_audio_output_get_state(o.ptr, &state))
	return uint32(state), code
}

func (o *AudioOutput) Close() int32 {
	code := int32(C.ti_audio_output_destroy(o.ptr))
	if code == 0 {
		o.ptr = nil
		o.handle.Delete()
	}
	return code
}

func NewVideoOutput(decoder, strategy uint32, watermark *time.Duration, callbacks OutputCallbacks) (*VideoOutput, int32) {
	has, value := buffer(strategy, watermark)
	context := &outputContext{callbacks: callbacks}
	handle := cgo.NewHandle(context)
	var pointer *C.TiVideoOutput
	code := int32(C.ti_go_video_output_create(C.TiVideoDecoderPreference(decoder), C.TiOutputBufferStrategy(strategy), has, value, C.uintptr_t(handle), &pointer))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &VideoOutput{ptr: pointer, handle: handle}, 0
}
func (o *VideoOutput) Attach(c *Conn, id uint8) int32 {
	return int32(C.tirtc_video_output_attach(o.ptr, c.ptr, C.uint8_t(id)))
}
func (o *VideoOutput) Detach() int32 { return int32(C.tirtc_video_output_detach(o.ptr)) }
func (o *VideoOutput) State() (uint32, int32) {
	var state C.TiOutputState
	code := int32(C.ti_video_output_get_state(o.ptr, &state))
	return uint32(state), code
}
func (o *VideoOutput) Close() int32 {
	code := int32(C.ti_video_output_destroy(o.ptr))
	if code == 0 {
		o.ptr = nil
		o.handle.Delete()
	}
	return code
}

func NewEncodedAudioOutput(callbacks OutputCallbacks) (*EncodedAudioOutput, int32) {
	context := &outputContext{callbacks: callbacks}
	handle := cgo.NewHandle(context)
	var pointer *C.TiEncodedAudioOutput
	code := int32(C.ti_go_encoded_audio_output_create(C.uintptr_t(handle), &pointer))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &EncodedAudioOutput{pointer, handle}, 0
}
func (o *EncodedAudioOutput) Attach(c *Conn, id uint8) int32 {
	return int32(C.tirtc_encoded_audio_output_attach(o.ptr, c.ptr, C.uint8_t(id)))
}
func (o *EncodedAudioOutput) Detach() int32 { return int32(C.tirtc_encoded_audio_output_detach(o.ptr)) }
func (o *EncodedAudioOutput) State() (uint32, int32) {
	var state C.TiOutputState
	code := int32(C.ti_encoded_audio_output_get_state(o.ptr, &state))
	return uint32(state), code
}
func (o *EncodedAudioOutput) Close() int32 {
	code := int32(C.ti_encoded_audio_output_destroy(o.ptr))
	if code == 0 {
		o.ptr = nil
		o.handle.Delete()
	}
	return code
}

func NewEncodedVideoOutput(callbacks OutputCallbacks) (*EncodedVideoOutput, int32) {
	context := &outputContext{callbacks: callbacks}
	handle := cgo.NewHandle(context)
	var pointer *C.TiEncodedVideoOutput
	code := int32(C.ti_go_encoded_video_output_create(C.uintptr_t(handle), &pointer))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &EncodedVideoOutput{pointer, handle}, 0
}
func (o *EncodedVideoOutput) Attach(c *Conn, id uint8) int32 {
	return int32(C.tirtc_encoded_video_output_attach(o.ptr, c.ptr, C.uint8_t(id)))
}
func (o *EncodedVideoOutput) Detach() int32 { return int32(C.tirtc_encoded_video_output_detach(o.ptr)) }
func (o *EncodedVideoOutput) State() (uint32, int32) {
	var state C.TiOutputState
	code := int32(C.ti_encoded_video_output_get_state(o.ptr, &state))
	return uint32(state), code
}
func (o *EncodedVideoOutput) Close() int32 {
	code := int32(C.ti_encoded_video_output_destroy(o.ptr))
	if code == 0 {
		o.ptr = nil
		o.handle.Delete()
	}
	return code
}

type RecordingTask struct{ ptr *C.TiRtcRecordingTask }

func (c *Conn) StartRecording(video int32, audio int32) (*RecordingTask, int32) {
	options := C.TiRtcStartRecordingOptions{video_stream_id: C.int32_t(video), audio_stream_id: C.int32_t(audio)}
	var task *C.TiRtcRecordingTask
	code := int32(C.tirtc_conn_start_recording(c.ptr, &options, &task))
	if code != 0 {
		return nil, code
	}
	return &RecordingTask{task}, 0
}
func (t *RecordingTask) Stop() (string, int64, int32, bool) {
	var file C.TiRtcMp4File
	stopCode := int32(C.tirtc_recording_task_stop(t.ptr, &file))
	path := ""
	duration := int64(0)
	if stopCode == 0 {
		path = C.GoString(file.file_path)
		duration = int64(file.duration_ms)
	}
	destroy := int32(C.tirtc_recording_task_destroy(t.ptr))
	if destroy == 0 {
		t.ptr = nil
	}
	if stopCode != 0 {
		return "", 0, stopCode, destroy == 0
	}
	return path, duration, destroy, destroy == 0
}

func (o *VideoOutput) Snapshot() (string, int32) {
	var file C.TiVideoSnapshotFile
	code := int32(C.ti_video_output_take_snapshot(o.ptr, &file))
	if code != 0 || file.file_path == nil {
		return "", code
	}
	return C.GoString(file.file_path), 0
}

func boolByte(value bool) C.uint8_t {
	if value {
		return 1
	}
	return 0
}
func copyBytes(data *C.uint8_t, size C.uint64_t) []byte {
	if data == nil || size == 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(data), C.int(size))
}

//export goTiConnState
func goTiConnState(handle C.uintptr_t, state C.uint32_t, code C.int32_t) {
	context := cgo.Handle(handle).Value().(*connContext)
	context.emit(func(callbacks ConnCallbacks) {
		if callbacks.OnState != nil {
			callbacks.OnState(uint32(state), int32(code))
		}
	})
}

//export goTiConnCommand
func goTiConnCommand(handle C.uintptr_t, command C.uint32_t, data *C.uint8_t, size C.uint64_t) {
	context := cgo.Handle(handle).Value().(*connContext)
	payload := copyBytes(data, size)
	context.emit(func(callbacks ConnCallbacks) {
		if callbacks.OnCommand != nil {
			callbacks.OnCommand(uint32(command), payload)
		}
	})
}

//export goTiConnMessage
func goTiConnMessage(handle C.uintptr_t, stream C.uint8_t, timestamp C.uint32_t, data *C.uint8_t, size C.uint64_t) {
	context := cgo.Handle(handle).Value().(*connContext)
	payload := copyBytes(data, size)
	context.emit(func(callbacks ConnCallbacks) {
		if callbacks.OnMessage != nil {
			callbacks.OnMessage(uint8(stream), uint32(timestamp), payload)
		}
	})
}

//export goTiOutputState
func goTiOutputState(handle C.uintptr_t, state C.uint32_t) {
	context := cgo.Handle(handle).Value().(*outputContext)
	if context.callbacks.OnState != nil {
		context.callbacks.OnState(uint32(state))
	}
}

//export goTiOutputError
func goTiOutputError(handle C.uintptr_t, code C.int32_t, message *C.char) {
	context := cgo.Handle(handle).Value().(*outputContext)
	if context.callbacks.OnError != nil {
		context.callbacks.OnError(int32(code), C.GoString(message))
	}
}

//export goTiAudioFrame
func goTiAudioFrame(handle C.uintptr_t, frame *C.TiAudioFrame) {
	context := cgo.Handle(handle).Value().(*outputContext)
	var info C.TiAudioFrameInfo
	if context.callbacks.OnAudioFrame != nil && C.ti_audio_frame_get_info(frame, &info) == C.TI_ERROR_OK {
		context.callbacks.OnAudioFrame(AudioFrame{Data: copyBytes(info.data, info.data_size), PTSUs: int64(info.pts_us), SourceTimeUTCUs: int64(info.source_time_utc_us), HasSourceTime: info.has_source_time != 0, Format: uint32(info.format.sample_format), SampleRateHz: int(info.format.sample_rate_hz), Channels: int(info.format.channels), SamplesPerChannel: int(info.samples_per_channel), Discontinuity: info.discontinuity != 0})
	}
}

//export goTiVideoFrame
func goTiVideoFrame(handle C.uintptr_t, frame *C.TiVideoFrame) {
	context := cgo.Handle(handle).Value().(*outputContext)
	if context.callbacks.OnVideoFrame == nil {
		return
	}
	var info C.TiVideoFrameInfo
	if C.ti_video_frame_get_info(frame, &info) != C.TI_ERROR_OK {
		return
	}
	count := int(info.plane_count)
	planes := make([]VideoPlane, 0, count)
	for n := 0; n < count; n++ {
		plane := info.planes[n]
		planes = append(planes, VideoPlane{Stride: int(plane.stride_bytes), Data: copyBytes(plane.data, plane.data_size)})
	}
	context.callbacks.OnVideoFrame(VideoFrame{PTSUs: int64(info.pts_us), SourceTimeUTCUs: int64(info.source_time_utc_us), HasSourceTime: info.has_source_time != 0, PixelFormat: uint32(info.pixel_format), Width: int(info.width), Height: int(info.height), Planes: planes, Discontinuity: info.discontinuity != 0})
}

//export goTiEncodedAudioFrame
func goTiEncodedAudioFrame(handle C.uintptr_t, frame *C.TiEncodedAudioFrame) {
	context := cgo.Handle(handle).Value().(*outputContext)
	var info C.TiEncodedAudioFrameInfo
	if context.callbacks.OnEncodedAudio != nil && C.ti_encoded_audio_frame_get_info(frame, &info) == C.TI_ERROR_OK {
		context.callbacks.OnEncodedAudio(EncodedAudioFrame{Data: copyBytes(info.data, info.data_size), CodecConfig: copyBytes(info.codec_config, info.codec_config_size), PTSUs: int64(info.pts_us), SourceTimeUTCUs: int64(info.source_time_utc_us), HasSourceTime: info.has_source_time != 0, Codec: uint32(info.codec), Bitstream: uint32(info.bitstream_format), SampleRateHz: int(info.sample_rate_hz), Channels: int(info.channels), Discontinuity: info.discontinuity != 0})
	}
}

//export goTiEncodedVideoFrame
func goTiEncodedVideoFrame(handle C.uintptr_t, frame *C.TiEncodedVideoFrame) {
	context := cgo.Handle(handle).Value().(*outputContext)
	var info C.TiEncodedVideoFrameInfo
	if context.callbacks.OnEncodedVideo != nil && C.ti_encoded_video_frame_get_info(frame, &info) == C.TI_ERROR_OK {
		context.callbacks.OnEncodedVideo(EncodedVideoFrame{Data: copyBytes(info.data, info.data_size), CodecConfig: copyBytes(info.codec_config, info.codec_config_size), PTSUs: int64(info.pts_us), SourceTimeUTCUs: int64(info.source_time_utc_us), HasSourceTime: info.has_source_time != 0, Codec: uint32(info.codec), Bitstream: uint32(info.bitstream_format), Width: int(info.width), Height: int(info.height), KeyFrame: info.key_frame != 0, Discontinuity: info.discontinuity != 0})
	}
}
