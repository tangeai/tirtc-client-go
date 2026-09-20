package native

/*
#include "bridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"
)

type RawDumpStopReason uint32

type RawDumpArchive struct {
	CaptureID, Path, SHA256              string
	Size, CapturedDurationMS             uint64
	CaptureComplete                      bool
	StopReason                           RawDumpStopReason
	UnsavedPacketCount, UnsavedByteCount uint64
	Empty                                bool
}

type RawDump struct{ ptr *C.TiRawDump }

func rawDumpIDs(values []uint8) (*C.uint8_t, C.size_t) {
	if len(values) == 0 {
		return nil, 0
	}
	return (*C.uint8_t)(C.CBytes(values)), C.size_t(len(values))
}

func (c *Conn) StartRawDump(audio, video []uint8) (*RawDump, int32) {
	audioPointer, audioCount := rawDumpIDs(audio)
	videoPointer, videoCount := rawDumpIDs(video)
	defer C.free(unsafe.Pointer(audioPointer))
	defer C.free(unsafe.Pointer(videoPointer))
	options := C.TiRtcRawDumpOptions{
		audio_stream_ids: audioPointer, audio_stream_id_count: audioCount,
		video_stream_ids: videoPointer, video_stream_id_count: videoCount,
		uplink_audio_stream_ids: nil, uplink_audio_stream_id_count: 0,
	}
	var dump *C.TiRawDump
	code := int32(C.tirtc_conn_start_raw_dump(c.ptr, &options, &dump))
	if code != 0 {
		return nil, code
	}
	return &RawDump{ptr: dump}, 0
}

func (r *CloudStorageReplay) StartRawDump(audio, video []uint8) (*RawDump, int32) {
	audioPointer, audioCount := rawDumpIDs(audio)
	videoPointer, videoCount := rawDumpIDs(video)
	defer C.free(unsafe.Pointer(audioPointer))
	defer C.free(unsafe.Pointer(videoPointer))
	options := C.TiCloudStorageRawDumpOptions{
		audio_channel_ids: audioPointer, audio_channel_id_count: audioCount,
		video_channel_ids: videoPointer, video_channel_id_count: videoCount,
	}
	var dump *C.TiRawDump
	code := int32(C.ti_cloud_storage_replay_start_raw_dump(r.ptr, &options, &dump))
	if code != 0 {
		return nil, code
	}
	return &RawDump{ptr: dump}, 0
}

func (d *RawDump) Stop() (RawDumpArchive, int32) {
	var value C.TiRawDumpArchive
	code := int32(C.ti_raw_dump_stop(d.ptr, &value))
	if code != 0 {
		return RawDumpArchive{}, code
	}
	return RawDumpArchive{
		CaptureID: C.GoString(value.capture_id), Path: C.GoString(value.file_path),
		Size: uint64(value.file_size), SHA256: C.GoString(value.sha256),
		CapturedDurationMS: uint64(value.captured_duration_ms), CaptureComplete: value.capture_complete != 0,
		StopReason: RawDumpStopReason(value.stop_reason), UnsavedPacketCount: uint64(value.unsaved_packet_count),
		UnsavedByteCount: uint64(value.unsaved_byte_count), Empty: value.empty != 0,
	}, 0
}

func (d *RawDump) Close() int32 {
	code := int32(C.ti_raw_dump_destroy(d.ptr))
	if code == 0 {
		d.ptr = nil
	}
	return code
}
