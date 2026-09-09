#ifndef TI_CLOUD_STORAGE_H_
#define TI_CLOUD_STORAGE_H_

#include <stddef.h>
#include <stdint.h>

#include "ti/error.h"
#include "ti/media.h"
#include "ti/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct TiCloudStorage TiCloudStorage;
typedef struct TiCloudStorageClient TiCloudStorageClient;
typedef struct TiCloudStorageRecordingRequest TiCloudStorageRecordingRequest;
typedef struct TiCloudStorageRecordingDaysRequest TiCloudStorageRecordingDaysRequest;
typedef struct TiCloudStorageReplay TiCloudStorageReplay;
typedef struct TiCloudStorageRecordingTask TiCloudStorageRecordingTask;
typedef struct TiCloudStorageExportTask TiCloudStorageExportTask;

typedef struct TiCloudStorageInitOptions {
  const char* app_id;
  const char* endpoint;
  const char* cache_root_dir;
  uint8_t console_log_enabled;
} TiCloudStorageInitOptions;

#define TI_CLOUD_STORAGE_INIT_OPTIONS_INITIALIZER {NULL, NULL, NULL, 0u}

/**
 * Native-managed Cloud Storage client configuration.
 *
 * The client copies all strings. Access keys remain inside Runtime and are used only to obtain
 * short-lived per-device read credentials. Multiple clients may use different application
 * identities in one process; cache_root_dir and console_log_enabled are process-wide and must
 * agree with other active Runtime products.
 */
typedef struct TiCloudStorageClientOptions {
  const char* app_id;
  const char* access_key_id;
  const char* access_key_secret;
  const char* endpoint;
  const char* cache_root_dir;
  uint8_t console_log_enabled;
} TiCloudStorageClientOptions;

#define TI_CLOUD_STORAGE_CLIENT_OPTIONS_INITIALIZER {NULL, NULL, NULL, NULL, NULL, 0u}

typedef struct TiCloudStorageRecordingRange {
  int64_t start_time_ms;
  int64_t end_time_ms;
} TiCloudStorageRecordingRange;

typedef uint32_t TiCloudStorageRecordingTrackKind;
#define TI_CLOUD_STORAGE_RECORDING_TRACK_VIDEO ((TiCloudStorageRecordingTrackKind)1)
#define TI_CLOUD_STORAGE_RECORDING_TRACK_AUDIO ((TiCloudStorageRecordingTrackKind)2)

typedef struct TiCloudStorageRecordingTrack {
  TiCloudStorageRecordingTrackKind kind;
  uint8_t channel_id;
} TiCloudStorageRecordingTrack;

typedef uint32_t TiCloudStorageRecordingGapReason;
#define TI_CLOUD_STORAGE_RECORDING_GAP_UNKNOWN ((TiCloudStorageRecordingGapReason)0)
#define TI_CLOUD_STORAGE_RECORDING_GAP_NOT_FOUND ((TiCloudStorageRecordingGapReason)1)
#define TI_CLOUD_STORAGE_RECORDING_GAP_DOWNLOAD_FAILED ((TiCloudStorageRecordingGapReason)2)
#define TI_CLOUD_STORAGE_RECORDING_GAP_INTEGRITY_FAILED ((TiCloudStorageRecordingGapReason)3)
#define TI_CLOUD_STORAGE_RECORDING_GAP_MEDIA_UNREADABLE ((TiCloudStorageRecordingGapReason)4)
#define TI_CLOUD_STORAGE_RECORDING_GAP_NO_KEY_FRAME ((TiCloudStorageRecordingGapReason)5)
#define TI_CLOUD_STORAGE_RECORDING_GAP_NO_RECORDING ((TiCloudStorageRecordingGapReason)6)
#define TI_CLOUD_STORAGE_RECORDING_GAP_DECRYPTION_FAILED ((TiCloudStorageRecordingGapReason)7)
#define TI_CLOUD_STORAGE_RECORDING_GAP_UNSUPPORTED_MEDIA ((TiCloudStorageRecordingGapReason)8)
#define TI_CLOUD_STORAGE_RECORDING_GAP_TRACK_UNAVAILABLE ((TiCloudStorageRecordingGapReason)9)

typedef struct TiCloudStorageRecordingGap {
  TiCloudStorageRecordingRange range;
  const TiCloudStorageRecordingTrack* tracks;
  size_t track_count;
  const TiCloudStorageRecordingGapReason* reasons;
  size_t reason_count;
} TiCloudStorageRecordingGap;

#define TI_CLOUD_STORAGE_RECORDING_DATE_SIZE 11u

typedef struct TiCloudStorageRecordingDay {
  char date[TI_CLOUD_STORAGE_RECORDING_DATE_SIZE];
  uint8_t has_recording;
} TiCloudStorageRecordingDay;

typedef void(TI_CALL* TiCloudStorageRecordingRequestOnCompletedFn)(
    TiCloudStorageRecordingRequest* request, void* user_data);

typedef struct TiCloudStorageRecordingRequestCallbacks {
  TiCloudStorageRecordingRequestOnCompletedFn on_completed;
  TiCallbackDispatcher dispatcher;
} TiCloudStorageRecordingRequestCallbacks;

#define TI_CLOUD_STORAGE_RECORDING_REQUEST_CALLBACKS_INITIALIZER \
  {NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiCloudStorageRecordingDaysRequestOnCompletedFn)(
    TiCloudStorageRecordingDaysRequest* request, void* user_data);

typedef struct TiCloudStorageRecordingDaysRequestCallbacks {
  TiCloudStorageRecordingDaysRequestOnCompletedFn on_completed;
  TiCallbackDispatcher dispatcher;
} TiCloudStorageRecordingDaysRequestCallbacks;

#define TI_CLOUD_STORAGE_RECORDING_DAYS_REQUEST_CALLBACKS_INITIALIZER \
  {NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef uint32_t TiCloudStorageReplaySpeed;
#define TI_CLOUD_STORAGE_REPLAY_SPEED_1X ((TiCloudStorageReplaySpeed)0)
#define TI_CLOUD_STORAGE_REPLAY_SPEED_2X ((TiCloudStorageReplaySpeed)1)
#define TI_CLOUD_STORAGE_REPLAY_SPEED_4X ((TiCloudStorageReplaySpeed)2)
#define TI_CLOUD_STORAGE_REPLAY_SPEED_8X ((TiCloudStorageReplaySpeed)3)
#define TI_CLOUD_STORAGE_REPLAY_SPEED_0_5X ((TiCloudStorageReplaySpeed)4)
#define TI_CLOUD_STORAGE_REPLAY_SPEED_0_25X ((TiCloudStorageReplaySpeed)5)
#define TI_CLOUD_STORAGE_REPLAY_SPEED_0_125X ((TiCloudStorageReplaySpeed)6)

// Periodic playback-position notification. Intermediate positions may be coalesced; call
// ti_cloud_storage_replay_get_current_time_ms() when the latest position is required synchronously.
typedef void(TI_CALL* TiCloudStorageReplayOnTimeChangedFn)(TiCloudStorageReplay* replay,
                                                           int64_t time_ms, void* user_data);
typedef void(TI_CALL* TiCloudStorageReplayOnCompletedFn)(TiCloudStorageReplay* replay,
                                                         void* user_data);
typedef void(TI_CALL* TiCloudStorageReplayOnErrorFn)(TiCloudStorageReplay* replay, TiError error,
                                                     void* user_data);
typedef void(TI_CALL* TiCloudStorageReplayOnRecordingGapFn)(TiCloudStorageReplay* replay,
                                                            const TiCloudStorageRecordingGap* gap,
                                                            void* user_data);

typedef struct TiCloudStorageReplayCallbacks {
  TiCloudStorageReplayOnTimeChangedFn on_time_changed;
  TiCloudStorageReplayOnCompletedFn on_completed;
  TiCloudStorageReplayOnErrorFn on_error;
  TiCloudStorageReplayOnRecordingGapFn on_recording_gap;
  TiCallbackDispatcher dispatcher;
} TiCloudStorageReplayCallbacks;

#define TI_CLOUD_STORAGE_REPLAY_CALLBACKS_INITIALIZER \
  {NULL, NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef struct TiCloudStorageMp4File {
  const char* file_path;
  int64_t duration_ms;
} TiCloudStorageMp4File;

#define TI_CLOUD_STORAGE_CHANNEL_ID_NONE ((int32_t)-1)

typedef struct TiCloudStorageStartRecordingOptions {
  int32_t video_channel_id;
  int32_t audio_channel_id;
} TiCloudStorageStartRecordingOptions;

#define TI_CLOUD_STORAGE_START_RECORDING_OPTIONS_INITIALIZER \
  {TI_CLOUD_STORAGE_CHANNEL_ID_NONE, TI_CLOUD_STORAGE_CHANNEL_ID_NONE}

typedef struct TiCloudStorageExportOptions {
  int64_t start_time_ms;
  int64_t end_time_ms;
  int32_t video_channel_id;
  int32_t audio_channel_id;
} TiCloudStorageExportOptions;

typedef struct TiCloudStorageExportProgress {
  double fraction;
  int64_t covered_duration_ms;
} TiCloudStorageExportProgress;

typedef struct TiCloudStorageExportSegment {
  TiCloudStorageRecordingRange source_range;
  int64_t output_start_ms;
  int64_t output_end_ms;
} TiCloudStorageExportSegment;

typedef uint32_t TiCloudStorageExportTermination;
#define TI_CLOUD_STORAGE_EXPORT_TERMINATION_EXHAUSTED ((TiCloudStorageExportTermination)0)
#define TI_CLOUD_STORAGE_EXPORT_TERMINATION_INTERRUPTED ((TiCloudStorageExportTermination)1)
#define TI_CLOUD_STORAGE_EXPORT_TERMINATION_CANCELLED ((TiCloudStorageExportTermination)2)
#define TI_CLOUD_STORAGE_EXPORT_TERMINATION_FAILED ((TiCloudStorageExportTermination)3)

typedef struct TiCloudStorageExportReport {
  TiCloudStorageRecordingRange requested_range;
  int64_t covered_duration_ms;
  size_t segment_count;
  size_t gap_count;
  size_t unprocessed_range_count;
  uint8_t complete;
  TiCloudStorageExportTermination termination;
  TiError cause;
} TiCloudStorageExportReport;

typedef void(TI_CALL* TiCloudStorageExportOnProgressFn)(TiCloudStorageExportTask* task,
                                                        double progress, void* user_data);
typedef void(TI_CALL* TiCloudStorageExportOnProgressDetailFn)(
    TiCloudStorageExportTask* task, const TiCloudStorageExportProgress* progress, void* user_data);
typedef void(TI_CALL* TiCloudStorageExportOnRecordingGapFn)(TiCloudStorageExportTask* task,
                                                            const TiCloudStorageRecordingGap* gap,
                                                            void* user_data);
typedef void(TI_CALL* TiCloudStorageExportOnCompletedFn)(TiCloudStorageExportTask* task,
                                                         TiError error,
                                                         const TiCloudStorageMp4File* file,
                                                         void* user_data);

typedef struct TiCloudStorageExportCallbacks {
  TiCloudStorageExportOnProgressFn on_progress;
  TiCloudStorageExportOnProgressDetailFn on_progress_detail;
  TiCloudStorageExportOnRecordingGapFn on_recording_gap;
  TiCloudStorageExportOnCompletedFn on_completed;
  TiCallbackDispatcher dispatcher;
} TiCloudStorageExportCallbacks;

#define TI_CLOUD_STORAGE_EXPORT_OPTIONS_INITIALIZER \
  {0, 0, TI_CLOUD_STORAGE_CHANNEL_ID_NONE, TI_CLOUD_STORAGE_CHANNEL_ID_NONE}
#define TI_CLOUD_STORAGE_EXPORT_CALLBACKS_INITIALIZER \
  {NULL, NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

#define TI_CLOUD_STORAGE_ERROR_RECORDING_UNREADABLE ((TiError)6122)
#define TI_CLOUD_STORAGE_ERROR_UNAVAILABLE ((TiError)6123)
#define TI_CLOUD_STORAGE_ERROR_STOPPED ((TiError)6124)
#define TI_CLOUD_STORAGE_ERROR_RECORDING_NOT_FOUND ((TiError)6134)
#define TI_CLOUD_STORAGE_ERROR_RECORDING_DOWNLOAD_FAILED ((TiError)6135)

TI_API TiError TI_CALL ti_cloud_storage_init(const TiCloudStorageInitOptions* options);
TI_API TiError TI_CALL ti_cloud_storage_uninit(void);
TI_API TiError TI_CALL ti_cloud_storage_client_create(const TiCloudStorageClientOptions* options,
                                                      TiCloudStorageClient** out_client);
TI_API TiError TI_CALL ti_cloud_storage_client_open_device(TiCloudStorageClient* client,
                                                           const char* device_id,
                                                           TiCloudStorage** out_cloud_storage);
TI_API TiError TI_CALL ti_cloud_storage_client_destroy(TiCloudStorageClient* client);
TI_API TiError TI_CALL ti_cloud_storage_create(const char* token,
                                               TiCloudStorage** out_cloud_storage);
TI_API TiError TI_CALL ti_cloud_storage_update_token(TiCloudStorage* cloud_storage,
                                                     const char* token);
TI_API TiError TI_CALL ti_cloud_storage_destroy(TiCloudStorage* cloud_storage);

TI_API TiError TI_CALL ti_cloud_storage_list_recordings(
    TiCloudStorage* cloud_storage, int64_t start_time_ms, int64_t end_time_ms,
    const TiCloudStorageRecordingRequestCallbacks* callbacks, void* user_data,
    TiCloudStorageRecordingRequest** out_request);
TI_API TiError TI_CALL
ti_cloud_storage_recording_request_cancel(TiCloudStorageRecordingRequest* request);
TI_API TiError TI_CALL ti_cloud_storage_recording_request_get_error(
    const TiCloudStorageRecordingRequest* request, TiError* out_error);
TI_API TiError TI_CALL ti_cloud_storage_recording_request_get_count(
    const TiCloudStorageRecordingRequest* request, size_t* out_count);
TI_API TiError TI_CALL ti_cloud_storage_recording_request_get_recording(
    const TiCloudStorageRecordingRequest* request, size_t index,
    TiCloudStorageRecordingRange* out_recording);
TI_API TiError TI_CALL
ti_cloud_storage_recording_request_destroy(TiCloudStorageRecordingRequest* request);

TI_API TiError TI_CALL ti_cloud_storage_list_recording_days(
    TiCloudStorage* cloud_storage, const char* start_date, const char* end_date,
    const char* time_zone_id, const TiCloudStorageRecordingDaysRequestCallbacks* callbacks,
    void* user_data, TiCloudStorageRecordingDaysRequest** out_request);
TI_API TiError TI_CALL
ti_cloud_storage_recording_days_request_cancel(TiCloudStorageRecordingDaysRequest* request);
TI_API TiError TI_CALL ti_cloud_storage_recording_days_request_get_error(
    const TiCloudStorageRecordingDaysRequest* request, TiError* out_error);
TI_API TiError TI_CALL ti_cloud_storage_recording_days_request_get_count(
    const TiCloudStorageRecordingDaysRequest* request, size_t* out_count);
TI_API TiError TI_CALL
ti_cloud_storage_recording_days_request_get_day(const TiCloudStorageRecordingDaysRequest* request,
                                                size_t index, TiCloudStorageRecordingDay* out_day);
TI_API TiError TI_CALL
ti_cloud_storage_recording_days_request_destroy(TiCloudStorageRecordingDaysRequest* request);

TI_API TiError TI_CALL ti_cloud_storage_replay_create(TiCloudStorage* cloud_storage,
                                                      TiCloudStorageReplay** out_replay);
TI_API TiError TI_CALL ti_cloud_storage_replay_set_callbacks(
    TiCloudStorageReplay* replay, const TiCloudStorageReplayCallbacks* callbacks, void* user_data);
TI_API TiError TI_CALL ti_cloud_storage_replay_play(TiCloudStorageReplay* replay,
                                                    int64_t start_time_ms, int64_t end_time_ms);
TI_API TiError TI_CALL ti_cloud_storage_replay_play_at(TiCloudStorageReplay* replay,
                                                       int64_t start_time_ms, int64_t end_time_ms,
                                                       int64_t initial_time_ms);
TI_API TiError TI_CALL ti_cloud_storage_replay_pause(TiCloudStorageReplay* replay);
TI_API TiError TI_CALL ti_cloud_storage_replay_resume(TiCloudStorageReplay* replay);
TI_API TiError TI_CALL ti_cloud_storage_replay_seek(TiCloudStorageReplay* replay, int64_t time_ms);
TI_API TiError TI_CALL ti_cloud_storage_replay_set_speed(TiCloudStorageReplay* replay,
                                                         TiCloudStorageReplaySpeed speed);
TI_API TiError TI_CALL ti_cloud_storage_replay_get_speed(const TiCloudStorageReplay* replay,
                                                         TiCloudStorageReplaySpeed* out_speed);
TI_API TiError TI_CALL ti_cloud_storage_replay_get_current_time_ms(
    const TiCloudStorageReplay* replay, uint8_t* out_has_value, int64_t* out_time_ms);
TI_API TiError TI_CALL ti_cloud_storage_replay_stop(TiCloudStorageReplay* replay);
TI_API TiError TI_CALL ti_cloud_storage_replay_destroy(TiCloudStorageReplay* replay);

TI_API TiError TI_CALL ti_cloud_storage_audio_output_attach(TiAudioOutput* output,
                                                            TiCloudStorageReplay* replay,
                                                            uint8_t channel_id);
TI_API TiError TI_CALL ti_cloud_storage_audio_output_detach(TiAudioOutput* output);
TI_API TiError TI_CALL ti_cloud_storage_video_output_attach(TiVideoOutput* output,
                                                            TiCloudStorageReplay* replay,
                                                            uint8_t channel_id);
TI_API TiError TI_CALL ti_cloud_storage_video_output_detach(TiVideoOutput* output);
TI_API TiError TI_CALL ti_cloud_storage_encoded_audio_output_attach(TiEncodedAudioOutput* output,
                                                                    TiCloudStorageReplay* replay,
                                                                    uint8_t channel_id);
TI_API TiError TI_CALL ti_cloud_storage_encoded_audio_output_detach(TiEncodedAudioOutput* output);
TI_API TiError TI_CALL ti_cloud_storage_encoded_video_output_attach(TiEncodedVideoOutput* output,
                                                                    TiCloudStorageReplay* replay,
                                                                    uint8_t channel_id);
TI_API TiError TI_CALL ti_cloud_storage_encoded_video_output_detach(TiEncodedVideoOutput* output);

TI_API TiError TI_CALL ti_cloud_storage_replay_start_recording(
    TiCloudStorageReplay* replay, const TiCloudStorageStartRecordingOptions* options,
    TiCloudStorageRecordingTask** out_task);
TI_API TiError TI_CALL ti_cloud_storage_recording_task_stop(TiCloudStorageRecordingTask* task,
                                                            TiCloudStorageMp4File* out_file);
TI_API TiError TI_CALL ti_cloud_storage_recording_task_destroy(TiCloudStorageRecordingTask* task);

TI_API TiError TI_CALL ti_cloud_storage_export_recording(
    TiCloudStorage* cloud_storage, const TiCloudStorageExportOptions* options,
    const TiCloudStorageExportCallbacks* callbacks, void* user_data,
    TiCloudStorageExportTask** out_task);
TI_API TiError TI_CALL ti_cloud_storage_export_task_get_progress(
    const TiCloudStorageExportTask* task, double* out_progress);
TI_API TiError TI_CALL ti_cloud_storage_export_task_get_progress_detail(
    const TiCloudStorageExportTask* task, TiCloudStorageExportProgress* out_progress);
TI_API TiError TI_CALL ti_cloud_storage_export_task_cancel(TiCloudStorageExportTask* task);
TI_API TiError TI_CALL ti_cloud_storage_export_task_wait(TiCloudStorageExportTask* task,
                                                         TiCloudStorageMp4File* out_file);
TI_API TiError TI_CALL ti_cloud_storage_export_task_get_report(
    const TiCloudStorageExportTask* task, TiCloudStorageExportReport* out_report);
TI_API TiError TI_CALL ti_cloud_storage_export_task_get_segment(
    const TiCloudStorageExportTask* task, size_t index, TiCloudStorageExportSegment* out_segment);
TI_API TiError TI_CALL ti_cloud_storage_export_task_get_gap(const TiCloudStorageExportTask* task,
                                                            size_t index,
                                                            TiCloudStorageRecordingGap* out_gap);
TI_API TiError TI_CALL ti_cloud_storage_export_task_get_gap_track(
    const TiCloudStorageExportTask* task, size_t gap_index, size_t track_index,
    TiCloudStorageRecordingTrack* out_track);
TI_API TiError TI_CALL ti_cloud_storage_export_task_get_gap_reason(
    const TiCloudStorageExportTask* task, size_t gap_index, size_t reason_index,
    TiCloudStorageRecordingGapReason* out_reason);
TI_API TiError TI_CALL ti_cloud_storage_export_task_get_unprocessed_range(
    const TiCloudStorageExportTask* task, size_t index, TiCloudStorageRecordingRange* out_range);
TI_API TiError TI_CALL ti_cloud_storage_export_task_stop(TiCloudStorageExportTask* task,
                                                         TiCloudStorageMp4File* out_file);
TI_API TiError TI_CALL ti_cloud_storage_export_task_destroy(TiCloudStorageExportTask* task);

#ifdef __cplusplus
}
#endif

#endif  // TI_CLOUD_STORAGE_H_
