#include "bridge.h"

#include <stddef.h>

extern void goTiConnState(uintptr_t context, uint32_t state, int32_t error);
extern void goTiConnCommand(uintptr_t context, uint32_t command, const uint8_t* data,
                            uint64_t size);
extern void goTiConnMessage(uintptr_t context, uint8_t stream_id, uint32_t timestamp_ms,
                            const uint8_t* data, uint64_t size);
extern void goTiOutputState(uintptr_t context, uint32_t state);
extern void goTiOutputError(uintptr_t context, int32_t error, const char* message);
extern void goTiAudioFrame(uintptr_t context, const TiAudioFrame* frame);
extern void goTiVideoFrame(uintptr_t context, const TiVideoFrame* frame);
extern void goTiEncodedAudioFrame(uintptr_t context, const TiEncodedAudioFrame* frame);
extern void goTiEncodedVideoFrame(uintptr_t context, const TiEncodedVideoFrame* frame);
extern void goTiCloudStorageListCompleted(uintptr_t context, TiCloudStorageRecordingRequest* request);
extern void goTiCloudStorageRecordingDaysCompleted(uintptr_t context,
                                            TiCloudStorageRecordingDaysRequest* request);
extern void goTiCloudStorageReplayTime(uintptr_t context, int64_t time_ms);
extern void goTiCloudStorageReplayCompleted(uintptr_t context);
extern void goTiCloudStorageReplayError(uintptr_t context, int32_t error);
extern void goTiCloudStorageExportProgress(uintptr_t context, double progress);
extern void goTiCloudStorageExportCompleted(uintptr_t context, int32_t error, const char* path,
                                     int64_t duration_ms);

static void conn_state(TiRtcConn* connection, TiRtcConnState state, TiError error,
                       void* user_data) {
  (void)connection;
  goTiConnState((uintptr_t)user_data, state, error);
}
static void conn_command(TiRtcConn* connection, uint32_t command, const uint8_t* data,
                         uint64_t size, void* user_data) {
  (void)connection;
  goTiConnCommand((uintptr_t)user_data, command, data, size);
}
static void conn_message(TiRtcConn* connection, uint8_t stream_id, uint32_t timestamp_ms,
                         const uint8_t* data, uint64_t size, void* user_data) {
  (void)connection;
  goTiConnMessage((uintptr_t)user_data, stream_id, timestamp_ms, data, size);
}
static TiRtcConnCallbacks conn_callbacks(void) {
  TiRtcConnCallbacks callbacks = TI_RTC_CONN_CALLBACKS_INITIALIZER;
  callbacks.on_state_changed = conn_state;
  callbacks.on_command = conn_command;
  callbacks.on_stream_message = conn_message;
  return callbacks;
}

TiError ti_go_init(const char* app_id, const char* endpoint, const char* cache_dir,
                   uint8_t console_log_enabled) {
  TiRtcInitOptions options = TI_RTC_INIT_OPTIONS_INITIALIZER;
  options.app_id = app_id;
  options.endpoint = endpoint;
  options.cache_root_dir = cache_dir;
  options.console_log_enabled = console_log_enabled;
  return tirtc_init(&options);
}

TiError ti_go_logging_upload(char* out_log_id, uint32_t capacity) {
  return ti_logging_upload(out_log_id, capacity);
}

TiError ti_go_cloud_storage_init(const char* app_id, const char* endpoint, const char* cache_dir,
                         uint8_t console_log_enabled) {
  TiCloudStorageInitOptions options = TI_CLOUD_STORAGE_INIT_OPTIONS_INITIALIZER;
  options.app_id = app_id;
  options.endpoint = endpoint;
  options.cache_root_dir = cache_dir;
  options.console_log_enabled = console_log_enabled;
  return ti_cloud_storage_init(&options);
}

static void cloud_storage_list_completed(TiCloudStorageRecordingRequest* request, void* user_data) {
  goTiCloudStorageListCompleted((uintptr_t)user_data, request);
}

TiError ti_go_cloud_storage_list(TiCloudStorage* cloud_storage, int64_t start_ms, int64_t end_ms, uintptr_t context,
                         TiCloudStorageRecordingRequest** out_request) {
  TiCloudStorageRecordingRequestCallbacks callbacks = TI_CLOUD_STORAGE_RECORDING_REQUEST_CALLBACKS_INITIALIZER;
  callbacks.on_completed = cloud_storage_list_completed;
  return ti_cloud_storage_list_recordings(cloud_storage, start_ms, end_ms, &callbacks, (void*)context, out_request);
}

static void cloud_storage_recording_days_completed(TiCloudStorageRecordingDaysRequest* request,
                                           void* user_data) {
  goTiCloudStorageRecordingDaysCompleted((uintptr_t)user_data, request);
}

TiError ti_go_cloud_storage_recording_days(TiCloudStorage* cloud_storage, const char* start_date,
                                   const char* end_date, const char* time_zone_id,
                                   uintptr_t context,
                                   TiCloudStorageRecordingDaysRequest** out_request) {
  TiCloudStorageRecordingDaysRequestCallbacks callbacks =
      TI_CLOUD_STORAGE_RECORDING_DAYS_REQUEST_CALLBACKS_INITIALIZER;
  callbacks.on_completed = cloud_storage_recording_days_completed;
  return ti_cloud_storage_list_recording_days(cloud_storage, start_date, end_date, time_zone_id, &callbacks,
                                     (void*)context, out_request);
}

static void cloud_storage_replay_time(TiCloudStorageReplay* replay, int64_t time_ms, void* user_data) {
  (void)replay;
  goTiCloudStorageReplayTime((uintptr_t)user_data, time_ms);
}
static void cloud_storage_replay_completed(TiCloudStorageReplay* replay, void* user_data) {
  (void)replay;
  goTiCloudStorageReplayCompleted((uintptr_t)user_data);
}
static void cloud_storage_replay_error(TiCloudStorageReplay* replay, TiError error, void* user_data) {
  (void)replay;
  goTiCloudStorageReplayError((uintptr_t)user_data, error);
}

TiError ti_go_cloud_storage_replay_create(TiCloudStorage* cloud_storage, uintptr_t context, TiCloudStorageReplay** out_replay) {
  TiError error = ti_cloud_storage_replay_create(cloud_storage, out_replay);
  if (error != TI_ERROR_OK) return error;
  TiCloudStorageReplayCallbacks callbacks = TI_CLOUD_STORAGE_REPLAY_CALLBACKS_INITIALIZER;
  callbacks.on_time_changed = cloud_storage_replay_time;
  callbacks.on_completed = cloud_storage_replay_completed;
  callbacks.on_error = cloud_storage_replay_error;
  error = ti_cloud_storage_replay_set_callbacks(*out_replay, &callbacks, (void*)context);
  if (error != TI_ERROR_OK) {
    (void)ti_cloud_storage_replay_destroy(*out_replay);
    *out_replay = NULL;
  }
  return error;
}

static void cloud_storage_export_progress(TiCloudStorageExportTask* task, double progress, void* user_data) {
  (void)task;
  goTiCloudStorageExportProgress((uintptr_t)user_data, progress);
}
static void cloud_storage_export_completed(TiCloudStorageExportTask* task, TiError error,
                                   const TiCloudStorageMp4File* file, void* user_data) {
  (void)task;
  goTiCloudStorageExportCompleted((uintptr_t)user_data, error,
                           file == NULL ? NULL : file->file_path,
                           file == NULL ? 0 : file->duration_ms);
}

TiError ti_go_cloud_storage_export(TiCloudStorage* cloud_storage, const TiCloudStorageExportOptions* options, uintptr_t context,
                           TiCloudStorageExportTask** out_task) {
  TiCloudStorageExportCallbacks callbacks = TI_CLOUD_STORAGE_EXPORT_CALLBACKS_INITIALIZER;
  callbacks.on_progress = cloud_storage_export_progress;
  callbacks.on_completed = cloud_storage_export_completed;
  return ti_cloud_storage_export_recording(cloud_storage, options, &callbacks, (void*)context, out_task);
}

TiError ti_go_conn_create(uintptr_t context, TiRtcConn** out_connection) {
  TiRtcConnCallbacks callbacks = conn_callbacks();
  TiRtcConnCreateOptions options = TI_RTC_CONN_CREATE_OPTIONS_INITIALIZER;
  options.callbacks = &callbacks;
  options.user_data = (void*)context;
  return tirtc_conn_create(&options, out_connection);
}

TiError ti_go_conn_send_command(TiRtcConn* connection, uint32_t command_id,
                                const uint8_t* data, uint64_t size) {
  TiRtcConnCommand command = {0};
  command.command = command_id;
  command.data = data;
  command.data_size = size;
  return tirtc_conn_send_command(connection, &command);
}

TiError ti_go_conn_send_message(TiRtcConn* connection, uint8_t stream_id,
                                uint32_t timestamp_ms, const uint8_t* data, uint64_t size) {
  TiRtcStreamMessage message = {0};
  message.timestamp_ms = timestamp_ms;
  message.data = data;
  message.data_size = size;
  return tirtc_conn_send_stream_message(connection, stream_id, &message);
}

static void output_state_audio(TiAudioOutput* output, TiOutputState state, void* user_data) {
  (void)output;
  goTiOutputState((uintptr_t)user_data, state);
}
static void output_error_audio(TiAudioOutput* output, TiError error, const char* message,
                               void* user_data) {
  (void)output;
  goTiOutputError((uintptr_t)user_data, error, message);
}
static void output_frame_audio(TiAudioOutput* output, const TiAudioFrame* frame, void* user_data) {
  (void)output;
  goTiAudioFrame((uintptr_t)user_data, frame);
}
static void output_state_video(TiVideoOutput* output, TiOutputState state, void* user_data) {
  (void)output;
  goTiOutputState((uintptr_t)user_data, state);
}
static void output_error_video(TiVideoOutput* output, TiError error, const char* message,
                               void* user_data) {
  (void)output;
  goTiOutputError((uintptr_t)user_data, error, message);
}
static void output_frame_video(TiVideoOutput* output, const TiVideoFrame* frame, void* user_data) {
  (void)output;
  goTiVideoFrame((uintptr_t)user_data, frame);
}
static void output_state_encoded_audio(TiEncodedAudioOutput* output, TiOutputState state,
                                       void* user_data) {
  (void)output;
  goTiOutputState((uintptr_t)user_data, state);
}
static void output_error_encoded_audio(TiEncodedAudioOutput* output, TiError error,
                                       const char* message, void* user_data) {
  (void)output;
  goTiOutputError((uintptr_t)user_data, error, message);
}
static void output_frame_encoded_audio(TiEncodedAudioOutput* output,
                                       const TiEncodedAudioFrame* frame, void* user_data) {
  (void)output;
  goTiEncodedAudioFrame((uintptr_t)user_data, frame);
}
static void output_state_encoded_video(TiEncodedVideoOutput* output, TiOutputState state,
                                       void* user_data) {
  (void)output;
  goTiOutputState((uintptr_t)user_data, state);
}
static void output_error_encoded_video(TiEncodedVideoOutput* output, TiError error,
                                       const char* message, void* user_data) {
  (void)output;
  goTiOutputError((uintptr_t)user_data, error, message);
}
static void output_frame_encoded_video(TiEncodedVideoOutput* output,
                                       const TiEncodedVideoFrame* frame, void* user_data) {
  (void)output;
  goTiEncodedVideoFrame((uintptr_t)user_data, frame);
}

static TiOutputBufferOptions buffer_options(TiOutputBufferStrategy strategy, uint8_t has_watermark,
                                            int32_t watermark_ms) {
  TiOutputBufferOptions options = TI_OUTPUT_BUFFER_OPTIONS_INITIALIZER;
  options.strategy = strategy;
  options.has_max_buffer_watermark_ms = has_watermark;
  options.max_buffer_watermark_ms = watermark_ms;
  return options;
}

TiError ti_go_audio_output_create(TiAudioAgcLevel agc, TiAudioAnsLevel ans,
                                  TiOutputBufferStrategy strategy, uint8_t has_watermark,
                                  int32_t watermark_ms, uintptr_t context,
                                  TiAudioOutput** out_output) {
  TiError status = ti_audio_output_create(out_output);
  if (status != TI_ERROR_OK) return status;
  TiAudioOutputOptions options = TI_AUDIO_OUTPUT_OPTIONS_INITIALIZER;
  options.agc_level = agc;
  options.ans_level = ans;
  TiOutputBufferOptions buffer = buffer_options(strategy, has_watermark, watermark_ms);
  status = ti_audio_output_set_options(*out_output, &options);
  if (status == TI_ERROR_OK) status = ti_audio_output_set_buffer_options(*out_output, &buffer);
  if (status == TI_ERROR_OK) {
    TiAudioOutputCallbacks callbacks = TI_AUDIO_OUTPUT_CALLBACKS_INITIALIZER;
    callbacks.on_frame = output_frame_audio;
    callbacks.on_state_changed = output_state_audio;
    callbacks.on_error = output_error_audio;
    status = ti_audio_output_set_callbacks(*out_output, &callbacks, (void*)context);
  }
  if (status != TI_ERROR_OK) {
    (void)ti_audio_output_destroy(*out_output);
    *out_output = NULL;
  }
  return status;
}

TiError ti_go_video_output_create(TiVideoDecoderPreference decoder,
                                  TiOutputBufferStrategy strategy, uint8_t has_watermark,
                                  int32_t watermark_ms, uintptr_t context,
                                  TiVideoOutput** out_output) {
  TiError status = ti_video_output_create(out_output);
  if (status != TI_ERROR_OK) return status;
  TiVideoOutputOptions options = TI_VIDEO_OUTPUT_OPTIONS_INITIALIZER;
  options.decoder_preference = decoder;
  TiOutputBufferOptions buffer = buffer_options(strategy, has_watermark, watermark_ms);
  status = ti_video_output_set_options(*out_output, &options);
  if (status == TI_ERROR_OK) status = ti_video_output_set_buffer_options(*out_output, &buffer);
  if (status == TI_ERROR_OK) {
    TiVideoOutputCallbacks callbacks = TI_VIDEO_OUTPUT_CALLBACKS_INITIALIZER;
    callbacks.on_frame = output_frame_video;
    callbacks.on_state_changed = output_state_video;
    callbacks.on_error = output_error_video;
    status = ti_video_output_set_callbacks(*out_output, &callbacks, (void*)context);
  }
  if (status != TI_ERROR_OK) {
    (void)ti_video_output_destroy(*out_output);
    *out_output = NULL;
  }
  return status;
}

TiError ti_go_encoded_audio_output_create(uintptr_t context,
                                          TiEncodedAudioOutput** out_output) {
  TiError status = ti_encoded_audio_output_create(out_output);
  if (status != TI_ERROR_OK) return status;
  TiEncodedAudioOutputCallbacks callbacks = TI_ENCODED_AUDIO_OUTPUT_CALLBACKS_INITIALIZER;
  callbacks.on_frame = output_frame_encoded_audio;
  callbacks.on_state_changed = output_state_encoded_audio;
  callbacks.on_error = output_error_encoded_audio;
  status = ti_encoded_audio_output_set_callbacks(*out_output, &callbacks, (void*)context);
  if (status != TI_ERROR_OK) {
    (void)ti_encoded_audio_output_destroy(*out_output);
    *out_output = NULL;
  }
  return status;
}

TiError ti_go_encoded_video_output_create(uintptr_t context,
                                          TiEncodedVideoOutput** out_output) {
  TiError status = ti_encoded_video_output_create(out_output);
  if (status != TI_ERROR_OK) return status;
  TiEncodedVideoOutputCallbacks callbacks = TI_ENCODED_VIDEO_OUTPUT_CALLBACKS_INITIALIZER;
  callbacks.on_frame = output_frame_encoded_video;
  callbacks.on_state_changed = output_state_encoded_video;
  callbacks.on_error = output_error_encoded_video;
  status = ti_encoded_video_output_set_callbacks(*out_output, &callbacks, (void*)context);
  if (status != TI_ERROR_OK) {
    (void)ti_encoded_video_output_destroy(*out_output);
    *out_output = NULL;
  }
  return status;
}
