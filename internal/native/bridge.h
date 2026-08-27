#ifndef TIRTC_GO_BRIDGE_H_
#define TIRTC_GO_BRIDGE_H_

#include <stdint.h>

#include "ti/media.h"
#include "ti/runtime.h"
#include "ti/rtc.h"
#include "ti/storage.h"

#ifdef __cplusplus
extern "C" {
#endif

TiError ti_go_init(const char* app_id, const char* endpoint, const char* cache_dir,
                   uint8_t console_log_enabled);
TiError ti_go_logging_upload(char* out_log_id, uint32_t capacity);

TiError ti_go_conn_create(uintptr_t context, TiRtcConn** out_connection);
TiError ti_go_conn_send_command(TiRtcConn* connection, uint32_t command,
                                const uint8_t* data, uint64_t size);
TiError ti_go_conn_send_message(TiRtcConn* connection, uint8_t stream_id,
                                uint32_t timestamp_ms, const uint8_t* data, uint64_t size);
TiError ti_go_audio_output_create(TiAudioAgcLevel agc, TiAudioAnsLevel ans,
                                  TiOutputBufferStrategy strategy, uint8_t has_watermark,
                                  int32_t watermark_ms, uintptr_t context,
                                  TiAudioOutput** out_output);
TiError ti_go_video_output_create(TiVideoDecoderPreference decoder,
                                  TiOutputBufferStrategy strategy, uint8_t has_watermark,
                                  int32_t watermark_ms, uintptr_t context,
                                  TiVideoOutput** out_output);
TiError ti_go_encoded_audio_output_create(uintptr_t context,
                                          TiEncodedAudioOutput** out_output);
TiError ti_go_encoded_video_output_create(uintptr_t context,
                                          TiEncodedVideoOutput** out_output);
TiError ti_go_cloud_storage_init(const char* app_id, const char* endpoint, const char* cache_dir,
                         uint8_t console_log_enabled);
TiError ti_go_cloud_storage_list(TiCloudStorage* cloud_storage, int64_t start_ms, int64_t end_ms, uintptr_t context,
                         TiCloudStorageRecordingRequest** out_request);
TiError ti_go_cloud_storage_recording_days(TiCloudStorage* cloud_storage, const char* start_date,
                                   const char* end_date, const char* time_zone_id,
                                   uintptr_t context,
                                   TiCloudStorageRecordingDaysRequest** out_request);
TiError ti_go_cloud_storage_replay_create(TiCloudStorage* cloud_storage, uintptr_t context, TiCloudStorageReplay** out_replay);
TiError ti_go_cloud_storage_export(TiCloudStorage* cloud_storage, const TiCloudStorageExportOptions* options, uintptr_t context,
                           TiCloudStorageExportTask** out_task);

#ifdef __cplusplus
}
#endif

#endif  // TIRTC_GO_BRIDGE_H_
