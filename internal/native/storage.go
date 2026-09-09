package native

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"runtime/cgo"
	"time"
	"unsafe"
)

const errorInUse int32 = 6026

func destroyAfterTerminal(destroy func() int32) int32 {
	return destroyAfterTerminalWithClock(destroy, time.Now, time.Sleep)
}

func destroyAfterTerminalWithClock(
	destroy func() int32,
	now func() time.Time,
	sleep func(time.Duration),
) int32 {
	deadline := now().Add(2 * time.Second)
	backoff := time.Millisecond
	for {
		code := destroy()
		if code != errorInUse {
			return code
		}
		if !now().Before(deadline) {
			return code
		}
		remaining := deadline.Sub(now())
		delay := backoff
		if delay > remaining {
			delay = remaining
		}
		sleep(delay)
		if backoff < 20*time.Millisecond {
			backoff *= 2
			if backoff > 20*time.Millisecond {
				backoff = 20 * time.Millisecond
			}
		}
	}
}

type CloudStorage struct {
	ptr *C.TiCloudStorage
}

type CloudStorageClient struct{ ptr *C.TiCloudStorageClient }

func NewCloudStorageClient(options ClientOptions) (*CloudStorageClient, int32) {
	app, freeApp := cString(options.AppID)
	defer freeApp()
	accessKeyID, freeAccessKeyID := cString(options.AccessKeyID)
	defer freeAccessKeyID()
	accessKeySecret, freeAccessKeySecret := cString(options.AccessKeySecret)
	defer freeAccessKeySecret()
	endpoint, freeEndpoint := cString(options.Endpoint)
	defer freeEndpoint()
	cacheDir, freeCacheDir := cString(options.CacheDir)
	defer freeCacheDir()
	var pointer *C.TiCloudStorageClient
	code := int32(C.ti_go_cloud_storage_client_create(app, accessKeyID, accessKeySecret, endpoint,
		cacheDir, boolByte(options.ConsoleLogEnabled), &pointer))
	if code != 0 {
		return nil, code
	}
	return &CloudStorageClient{ptr: pointer}, 0
}

func (c *CloudStorageClient) OpenDevice(deviceID string) (*CloudStorage, int32) {
	device, freeDevice := cString(deviceID)
	defer freeDevice()
	var pointer *C.TiCloudStorage
	code := int32(C.ti_cloud_storage_client_open_device(c.ptr, device, &pointer))
	if code != 0 {
		return nil, code
	}
	return &CloudStorage{ptr: pointer}, 0
}

func (c *CloudStorageClient) Close() int32 {
	code := int32(C.ti_cloud_storage_client_destroy(c.ptr))
	if code == 0 {
		c.ptr = nil
	}
	return code
}

func CloudStorageInit(options InitOptions) int32 {
	app, freeApp := cString(options.AppID)
	defer freeApp()
	endpoint, freeEndpoint := cString(options.Endpoint)
	defer freeEndpoint()
	cacheDir, freeCacheDir := cString(options.CacheDir)
	defer freeCacheDir()
	return int32(C.ti_go_cloud_storage_init(app, endpoint, cacheDir, boolByte(options.ConsoleLogEnabled)))
}

func CloudStorageShutdown() int32 { return int32(C.ti_cloud_storage_uninit()) }

func NewCloudStorage(token string) (*CloudStorage, int32) {
	value, freeValue := cString(token)
	defer freeValue()
	var pointer *C.TiCloudStorage
	code := int32(C.ti_cloud_storage_create(value, &pointer))
	if code != 0 {
		return nil, code
	}
	return &CloudStorage{ptr: pointer}, 0
}

func (s *CloudStorage) UpdateToken(token string) int32 {
	value, freeValue := cString(token)
	defer freeValue()
	return int32(C.ti_cloud_storage_update_token(s.ptr, value))
}

func (s *CloudStorage) Close() int32 {
	code := int32(C.ti_cloud_storage_destroy(s.ptr))
	if code == 0 {
		s.ptr = nil
	}
	return code
}

type CloudStorageRange struct{ StartMS, EndMS int64 }
type CloudStorageDay struct {
	Date         string
	HasRecording bool
}
type listResult struct{ code int32 }
type listContext struct{ done chan listResult }

type CloudStorageListRequest struct {
	ptr    *C.TiCloudStorageRecordingRequest
	handle cgo.Handle
	done   <-chan listResult
}

func (s *CloudStorage) StartList(startMS, endMS int64) (*CloudStorageListRequest, int32) {
	context := &listContext{done: make(chan listResult, 1)}
	handle := cgo.NewHandle(context)
	var request *C.TiCloudStorageRecordingRequest
	code := int32(C.ti_go_cloud_storage_list(s.ptr, C.int64_t(startMS), C.int64_t(endMS), C.uintptr_t(handle), &request))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &CloudStorageListRequest{ptr: request, handle: handle, done: context.done}, 0
}

func (r *CloudStorageListRequest) Wait() int32 { return (<-r.done).code }
func (r *CloudStorageListRequest) Cancel() int32 {
	return int32(C.ti_cloud_storage_recording_request_cancel(r.ptr))
}
func (r *CloudStorageListRequest) Ranges() ([]CloudStorageRange, int32) {
	var count C.size_t
	code := int32(C.ti_cloud_storage_recording_request_get_count(r.ptr, &count))
	ranges := make([]CloudStorageRange, 0, int(count))
	if code == 0 {
		for index := C.size_t(0); index < count; index++ {
			var value C.TiCloudStorageRecordingRange
			code = int32(C.ti_cloud_storage_recording_request_get_recording(r.ptr, index, &value))
			if code != 0 {
				break
			}
			ranges = append(ranges, CloudStorageRange{int64(value.start_time_ms), int64(value.end_time_ms)})
		}
	}
	return ranges, code
}
func (r *CloudStorageListRequest) Close() int32 {
	code := destroyAfterTerminal(func() int32 {
		return int32(C.ti_cloud_storage_recording_request_destroy(r.ptr))
	})
	if code == 0 {
		r.ptr = nil
		r.handle.Delete()
	}
	return code
}

func (s *CloudStorage) List(startMS, endMS int64) ([]CloudStorageRange, int32) {
	request, code := s.StartList(startMS, endMS)
	if code != 0 {
		return nil, code
	}
	if code = request.Wait(); code != 0 {
		_ = request.Close()
		return nil, code
	}
	ranges, code := request.Ranges()
	destroy := request.Close()
	if code == 0 {
		code = destroy
	}
	return ranges, code
}

type recordingDaysResult struct {
	code int32
	days []CloudStorageDay
}
type recordingDaysContext struct{ done chan recordingDaysResult }

type CloudStorageRecordingDaysRequest struct {
	ptr    *C.TiCloudStorageRecordingDaysRequest
	handle cgo.Handle
	done   <-chan recordingDaysResult
}

func (s *CloudStorage) StartRecordingDays(startDate, endDate, timeZoneID string) (*CloudStorageRecordingDaysRequest, int32) {
	context := &recordingDaysContext{done: make(chan recordingDaysResult, 1)}
	handle := cgo.NewHandle(context)
	start, freeStart := cString(startDate)
	defer freeStart()
	end, freeEnd := cString(endDate)
	defer freeEnd()
	timeZone, freeTimeZone := cString(timeZoneID)
	defer freeTimeZone()
	var request *C.TiCloudStorageRecordingDaysRequest
	code := int32(C.ti_go_cloud_storage_recording_days(
		s.ptr, start, end, timeZone, C.uintptr_t(handle), &request,
	))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &CloudStorageRecordingDaysRequest{ptr: request, handle: handle, done: context.done}, 0
}

func (r *CloudStorageRecordingDaysRequest) Wait() ([]CloudStorageDay, int32) {
	result := <-r.done
	return result.days, result.code
}
func (r *CloudStorageRecordingDaysRequest) Cancel() int32 {
	return int32(C.ti_cloud_storage_recording_days_request_cancel(r.ptr))
}
func (r *CloudStorageRecordingDaysRequest) Close() int32 {
	code := destroyAfterTerminal(func() int32 {
		return int32(C.ti_cloud_storage_recording_days_request_destroy(r.ptr))
	})
	if code == 0 {
		r.ptr = nil
		r.handle.Delete()
	}
	return code
}

type CloudStorageReplayCallbacks struct {
	Dispatch    func(func())
	OnTime      func(int64)
	OnCompleted func()
	OnError     func(int32)
	OnGap       func(CloudStorageRecordingGap)
}
type replayContext struct{ callbacks CloudStorageReplayCallbacks }
type CloudStorageReplay struct {
	ptr    *C.TiCloudStorageReplay
	handle cgo.Handle
}

func (s *CloudStorage) NewReplay(callbacks CloudStorageReplayCallbacks) (*CloudStorageReplay, int32) {
	handle := cgo.NewHandle(&replayContext{callbacks: callbacks})
	var pointer *C.TiCloudStorageReplay
	code := int32(C.ti_go_cloud_storage_replay_create(s.ptr, C.uintptr_t(handle), &pointer))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &CloudStorageReplay{ptr: pointer, handle: handle}, 0
}
func (r *CloudStorageReplay) Play(start, end, initial int64) int32 {
	return int32(C.ti_cloud_storage_replay_play_at(r.ptr, C.int64_t(start), C.int64_t(end), C.int64_t(initial)))
}
func (r *CloudStorageReplay) Pause() int32  { return int32(C.ti_cloud_storage_replay_pause(r.ptr)) }
func (r *CloudStorageReplay) Resume() int32 { return int32(C.ti_cloud_storage_replay_resume(r.ptr)) }
func (r *CloudStorageReplay) SeekTo(value int64) int32 {
	return int32(C.ti_cloud_storage_replay_seek(r.ptr, C.int64_t(value)))
}
func (r *CloudStorageReplay) SetSpeed(speed uint32) int32 {
	return int32(C.ti_cloud_storage_replay_set_speed(r.ptr, C.TiCloudStorageReplaySpeed(speed)))
}
func (r *CloudStorageReplay) Speed() (uint32, int32) {
	var value C.TiCloudStorageReplaySpeed
	code := int32(C.ti_cloud_storage_replay_get_speed(r.ptr, &value))
	return uint32(value), code
}
func (r *CloudStorageReplay) CurrentTime() (int64, bool, int32) {
	var has C.uint8_t
	var value C.int64_t
	code := int32(C.ti_cloud_storage_replay_get_current_time_ms(r.ptr, &has, &value))
	return int64(value), has != 0, code
}
func (r *CloudStorageReplay) Stop() int32 { return int32(C.ti_cloud_storage_replay_stop(r.ptr)) }
func (r *CloudStorageReplay) Close() int32 {
	code := int32(C.ti_cloud_storage_replay_destroy(r.ptr))
	if code == 0 {
		r.ptr = nil
		r.handle.Delete()
	}
	return code
}

func (o *AudioOutput) CloudStorageAttach(r *CloudStorageReplay, id uint8) int32 {
	return int32(C.ti_cloud_storage_audio_output_attach(o.ptr, r.ptr, C.uint8_t(id)))
}
func (o *AudioOutput) CloudStorageDetach() int32 {
	return int32(C.ti_cloud_storage_audio_output_detach(o.ptr))
}
func (o *VideoOutput) CloudStorageAttach(r *CloudStorageReplay, id uint8) int32 {
	return int32(C.ti_cloud_storage_video_output_attach(o.ptr, r.ptr, C.uint8_t(id)))
}
func (o *VideoOutput) CloudStorageDetach() int32 {
	return int32(C.ti_cloud_storage_video_output_detach(o.ptr))
}
func (o *EncodedAudioOutput) CloudStorageAttach(r *CloudStorageReplay, id uint8) int32 {
	return int32(C.ti_cloud_storage_encoded_audio_output_attach(o.ptr, r.ptr, C.uint8_t(id)))
}
func (o *EncodedAudioOutput) CloudStorageDetach() int32 {
	return int32(C.ti_cloud_storage_encoded_audio_output_detach(o.ptr))
}
func (o *EncodedVideoOutput) CloudStorageAttach(r *CloudStorageReplay, id uint8) int32 {
	return int32(C.ti_cloud_storage_encoded_video_output_attach(o.ptr, r.ptr, C.uint8_t(id)))
}
func (o *EncodedVideoOutput) CloudStorageDetach() int32 {
	return int32(C.ti_cloud_storage_encoded_video_output_detach(o.ptr))
}

type CloudStorageRecordingTask struct {
	ptr *C.TiCloudStorageRecordingTask
}

func stopAndDestroyCloudStorageRecordingTask(stop func() (string, int64, int32), destroy func() int32) (string, int64, int32, bool) {
	path, duration, code := stop()
	destroyCode := destroy()
	destroyed := destroyCode == 0
	if code != 0 {
		return "", 0, code, destroyed
	}
	return path, duration, destroyCode, destroyed
}

func (r *CloudStorageReplay) StartRecording(video, audio int32) (*CloudStorageRecordingTask, int32) {
	options := C.TiCloudStorageStartRecordingOptions{video_channel_id: C.int32_t(video), audio_channel_id: C.int32_t(audio)}
	var task *C.TiCloudStorageRecordingTask
	code := int32(C.ti_cloud_storage_replay_start_recording(r.ptr, &options, &task))
	if code != 0 {
		return nil, code
	}
	return &CloudStorageRecordingTask{task}, 0
}
func (t *CloudStorageRecordingTask) Stop() (string, int64, int32, bool) {
	if t == nil || t.ptr == nil {
		return "", 0, 6001, true
	}
	path, duration, code, destroyed := stopAndDestroyCloudStorageRecordingTask(func() (string, int64, int32) {
		var file C.TiCloudStorageMp4File
		code := int32(C.ti_cloud_storage_recording_task_stop(t.ptr, &file))
		if code != 0 {
			return "", 0, code
		}
		return C.GoString(file.file_path), int64(file.duration_ms), 0
	}, func() int32 {
		return int32(C.ti_cloud_storage_recording_task_destroy(t.ptr))
	})
	if destroyed {
		t.ptr = nil
	}
	return path, duration, code, destroyed
}

type CloudStorageExportCallbacks struct {
	OnProgress  func(CloudStorageExportProgress)
	OnGap       func(CloudStorageRecordingGap)
	OnCompleted func(int32, string, int64)
}
type exportContext struct {
	callbacks CloudStorageExportCallbacks
}
type CloudStorageExportTask struct {
	ptr    *C.TiCloudStorageExportTask
	handle cgo.Handle
}

type CloudStorageExportProgress struct {
	Fraction          float64
	CoveredDurationMS int64
}

type CloudStorageRecordingTrack struct {
	Kind      uint32
	ChannelID uint8
}

type CloudStorageRecordingGap struct {
	Range   CloudStorageRange
	Tracks  []CloudStorageRecordingTrack
	Reasons []uint32
}

type CloudStorageExportSegment struct {
	SourceRange                CloudStorageRange
	OutputStartMS, OutputEndMS int64
}

type CloudStorageExportReport struct {
	RequestedRange    CloudStorageRange
	CoveredDurationMS int64
	Segments          []CloudStorageExportSegment
	Gaps              []CloudStorageRecordingGap
	Unprocessed       []CloudStorageRange
	Complete          bool
	Termination       uint32
	Cause             int32
}

func (s *CloudStorage) Export(start, end int64, video, audio int32, callbacks CloudStorageExportCallbacks) (*CloudStorageExportTask, int32) {
	handle := cgo.NewHandle(&exportContext{callbacks})
	options := C.TiCloudStorageExportOptions{start_time_ms: C.int64_t(start), end_time_ms: C.int64_t(end), video_channel_id: C.int32_t(video), audio_channel_id: C.int32_t(audio)}
	var task *C.TiCloudStorageExportTask
	code := int32(C.ti_go_cloud_storage_export(s.ptr, &options, C.uintptr_t(handle), &task))
	if code != 0 {
		handle.Delete()
		return nil, code
	}
	return &CloudStorageExportTask{task, handle}, 0
}
func (t *CloudStorageExportTask) Progress() (CloudStorageExportProgress, int32) {
	var value C.TiCloudStorageExportProgress
	code := int32(C.ti_cloud_storage_export_task_get_progress_detail(t.ptr, &value))
	return CloudStorageExportProgress{Fraction: float64(value.fraction), CoveredDurationMS: int64(value.covered_duration_ms)}, code
}

func (t *CloudStorageExportTask) Cancel() int32 {
	return int32(C.ti_cloud_storage_export_task_cancel(t.ptr))
}

func (t *CloudStorageExportTask) Wait() (string, int64, int32) {
	var file C.TiCloudStorageMp4File
	code := int32(C.ti_cloud_storage_export_task_wait(t.ptr, &file))
	path := ""
	duration := int64(0)
	if code == 0 {
		path = C.GoString(file.file_path)
		duration = int64(file.duration_ms)
	}
	return path, duration, code
}

func exportGap(task *C.TiCloudStorageExportTask, index C.size_t) (CloudStorageRecordingGap, int32) {
	var value C.TiCloudStorageRecordingGap
	code := int32(C.ti_cloud_storage_export_task_get_gap(task, index, &value))
	result := CloudStorageRecordingGap{
		Range:   CloudStorageRange{StartMS: int64(value._range.start_time_ms), EndMS: int64(value._range.end_time_ms)},
		Tracks:  make([]CloudStorageRecordingTrack, 0, int(value.track_count)),
		Reasons: make([]uint32, 0, int(value.reason_count)),
	}
	for trackIndex := C.size_t(0); code == 0 && trackIndex < value.track_count; trackIndex++ {
		var track C.TiCloudStorageRecordingTrack
		code = int32(C.ti_cloud_storage_export_task_get_gap_track(task, index, trackIndex, &track))
		if code == 0 {
			result.Tracks = append(result.Tracks, CloudStorageRecordingTrack{Kind: uint32(track.kind), ChannelID: uint8(track.channel_id)})
		}
	}
	for reasonIndex := C.size_t(0); code == 0 && reasonIndex < value.reason_count; reasonIndex++ {
		var reason C.TiCloudStorageRecordingGapReason
		code = int32(C.ti_cloud_storage_export_task_get_gap_reason(task, index, reasonIndex, &reason))
		if code == 0 {
			result.Reasons = append(result.Reasons, uint32(reason))
		}
	}
	return result, code
}

func recordingGapFromC(value *C.TiCloudStorageRecordingGap) CloudStorageRecordingGap {
	result := CloudStorageRecordingGap{
		Range:   CloudStorageRange{StartMS: int64(value._range.start_time_ms), EndMS: int64(value._range.end_time_ms)},
		Tracks:  make([]CloudStorageRecordingTrack, 0, int(value.track_count)),
		Reasons: make([]uint32, 0, int(value.reason_count)),
	}
	if value.tracks != nil {
		for _, track := range unsafe.Slice(value.tracks, int(value.track_count)) {
			result.Tracks = append(result.Tracks, CloudStorageRecordingTrack{Kind: uint32(track.kind), ChannelID: uint8(track.channel_id)})
		}
	}
	if value.reasons != nil {
		for _, reason := range unsafe.Slice(value.reasons, int(value.reason_count)) {
			result.Reasons = append(result.Reasons, uint32(reason))
		}
	}
	return result
}

func (t *CloudStorageExportTask) Report() (CloudStorageExportReport, int32) {
	var value C.TiCloudStorageExportReport
	code := int32(C.ti_cloud_storage_export_task_get_report(t.ptr, &value))
	result := CloudStorageExportReport{
		RequestedRange:    CloudStorageRange{StartMS: int64(value.requested_range.start_time_ms), EndMS: int64(value.requested_range.end_time_ms)},
		CoveredDurationMS: int64(value.covered_duration_ms), Complete: value.complete != 0,
		Termination: uint32(value.termination), Cause: int32(value.cause),
		Segments:    make([]CloudStorageExportSegment, 0, int(value.segment_count)),
		Gaps:        make([]CloudStorageRecordingGap, 0, int(value.gap_count)),
		Unprocessed: make([]CloudStorageRange, 0, int(value.unprocessed_range_count)),
	}
	for index := C.size_t(0); code == 0 && index < value.segment_count; index++ {
		var segment C.TiCloudStorageExportSegment
		code = int32(C.ti_cloud_storage_export_task_get_segment(t.ptr, index, &segment))
		if code == 0 {
			result.Segments = append(result.Segments, CloudStorageExportSegment{
				SourceRange:   CloudStorageRange{StartMS: int64(segment.source_range.start_time_ms), EndMS: int64(segment.source_range.end_time_ms)},
				OutputStartMS: int64(segment.output_start_ms), OutputEndMS: int64(segment.output_end_ms),
			})
		}
	}
	for index := C.size_t(0); code == 0 && index < value.gap_count; index++ {
		var gap CloudStorageRecordingGap
		gap, code = exportGap(t.ptr, index)
		if code == 0 {
			result.Gaps = append(result.Gaps, gap)
		}
	}
	for index := C.size_t(0); code == 0 && index < value.unprocessed_range_count; index++ {
		var item C.TiCloudStorageRecordingRange
		code = int32(C.ti_cloud_storage_export_task_get_unprocessed_range(t.ptr, index, &item))
		if code == 0 {
			result.Unprocessed = append(result.Unprocessed, CloudStorageRange{StartMS: int64(item.start_time_ms), EndMS: int64(item.end_time_ms)})
		}
	}
	return result, code
}
func (t *CloudStorageExportTask) Close() int32 {
	code := destroyAfterTerminal(func() int32 {
		return int32(C.ti_cloud_storage_export_task_destroy(t.ptr))
	})
	if code == 0 {
		t.ptr = nil
		t.handle.Delete()
	}
	return code
}

//export goTiCloudStorageListCompleted
func goTiCloudStorageListCompleted(handle C.uintptr_t, request *C.TiCloudStorageRecordingRequest) {
	var code C.TiError
	if C.ti_cloud_storage_recording_request_get_error(request, &code) != 0 {
		code = C.TiError(6026)
	}
	cgo.Handle(handle).Value().(*listContext).done <- listResult{int32(code)}
}

//export goTiCloudStorageRecordingDaysCompleted
func goTiCloudStorageRecordingDaysCompleted(handle C.uintptr_t, request *C.TiCloudStorageRecordingDaysRequest) {
	var code C.TiError
	var count C.size_t
	if C.ti_cloud_storage_recording_days_request_get_error(request, &code) != 0 {
		code = C.TiError(6026)
	}
	days := []CloudStorageDay{}
	if code == 0 {
		code = C.TiError(C.ti_cloud_storage_recording_days_request_get_count(request, &count))
	}
	if code == 0 {
		days = make([]CloudStorageDay, 0, int(count))
		for index := C.size_t(0); index < count; index++ {
			var day C.TiCloudStorageRecordingDay
			code = C.TiError(C.ti_cloud_storage_recording_days_request_get_day(request, index, &day))
			if code != 0 {
				break
			}
			days = append(days, CloudStorageDay{
				Date:         C.GoString(&day.date[0]),
				HasRecording: day.has_recording != 0,
			})
		}
	}
	cgo.Handle(handle).Value().(*recordingDaysContext).done <- recordingDaysResult{
		code: int32(code),
		days: days,
	}
}

//export goTiCloudStorageReplayDispatch
func goTiCloudStorageReplayDispatch(handle C.uintptr_t, task C.TiCallbackTaskFn, data unsafe.Pointer) {
	context := cgo.Handle(handle).Value().(*replayContext)
	invoke := func() { C.ti_go_callback_task_run(task, data) }
	if context.callbacks.Dispatch != nil {
		context.callbacks.Dispatch(invoke)
	} else {
		invoke()
	}
}

//export goTiCloudStorageReplayTime
func goTiCloudStorageReplayTime(handle C.uintptr_t, value C.int64_t) {
	// Return through C so its callback gate always completes, even if a user handler panics.
	defer func() { _ = recover() }()
	callback := cgo.Handle(handle).Value().(*replayContext).callbacks.OnTime
	if callback != nil {
		callback(int64(value))
	}
}

//export goTiCloudStorageReplayCompleted
func goTiCloudStorageReplayCompleted(handle C.uintptr_t) {
	// Return through C so its callback gate always completes, even if a user handler panics.
	defer func() { _ = recover() }()
	callback := cgo.Handle(handle).Value().(*replayContext).callbacks.OnCompleted
	if callback != nil {
		callback()
	}
}

//export goTiCloudStorageReplayError
func goTiCloudStorageReplayError(handle C.uintptr_t, code C.int32_t) {
	// Return through C so its callback gate always completes, even if a user handler panics.
	defer func() { _ = recover() }()
	callback := cgo.Handle(handle).Value().(*replayContext).callbacks.OnError
	if callback != nil {
		callback(int32(code))
	}
}

//export goTiCloudStorageReplayGap
func goTiCloudStorageReplayGap(handle C.uintptr_t, value *C.TiCloudStorageRecordingGap) {
	// Return through C so its callback gate always completes, even if a user handler panics.
	defer func() { _ = recover() }()
	callback := cgo.Handle(handle).Value().(*replayContext).callbacks.OnGap
	if callback != nil {
		callback(recordingGapFromC(value))
	}
}

//export goTiCloudStorageExportProgressDetail
func goTiCloudStorageExportProgressDetail(handle C.uintptr_t, value C.double, covered C.int64_t) {
	callback := cgo.Handle(handle).Value().(*exportContext).callbacks.OnProgress
	if callback != nil {
		callback(CloudStorageExportProgress{Fraction: float64(value), CoveredDurationMS: int64(covered)})
	}
}

//export goTiCloudStorageExportGap
func goTiCloudStorageExportGap(handle C.uintptr_t, value *C.TiCloudStorageRecordingGap) {
	context := cgo.Handle(handle).Value().(*exportContext)
	gap := recordingGapFromC(value)
	if context.callbacks.OnGap != nil {
		context.callbacks.OnGap(gap)
	}
}

//export goTiCloudStorageExportCompleted
func goTiCloudStorageExportCompleted(handle C.uintptr_t, code C.int32_t, path *C.char, duration C.int64_t) {
	callback := cgo.Handle(handle).Value().(*exportContext).callbacks.OnCompleted
	if callback != nil {
		value := ""
		if path != nil {
			value = C.GoString(path)
		}
		callback(int32(code), value, int64(duration))
	}
}
