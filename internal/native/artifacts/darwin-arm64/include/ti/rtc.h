#ifndef TI_RTC_H_
#define TI_RTC_H_

#include <stddef.h>
#include <stdint.h>

#include "ti/error.h"
#include "ti/media.h"
#include "ti/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct TiRtcConnService TiRtcConnService;
typedef struct TiRtcConn TiRtcConn;
typedef struct TiRtcConnCallbacks TiRtcConnCallbacks;
typedef struct TiRtcRecordingTask TiRtcRecordingTask;

#define TI_RTC_STREAM_ID_NONE ((int32_t)-1)

typedef struct TiRtcMp4File {
  const char* file_path;
  int64_t duration_ms;
} TiRtcMp4File;

typedef struct TiRtcStartRecordingOptions {
  int32_t video_stream_id;
  int32_t audio_stream_id;
} TiRtcStartRecordingOptions;

#define TIRTC_START_RECORDING_OPTIONS_INITIALIZER {TI_RTC_STREAM_ID_NONE, TI_RTC_STREAM_ID_NONE}

typedef uint32_t TiRtcConnectLinkMode;
#define TI_RTC_CONNECT_LINK_MODE_AUTOMATIC ((TiRtcConnectLinkMode)0)
#define TI_RTC_CONNECT_LINK_MODE_DIRECT_ONLY ((TiRtcConnectLinkMode)1)
#define TI_RTC_CONNECT_LINK_MODE_RELAY_ONLY ((TiRtcConnectLinkMode)2)

typedef uint32_t TiRtcConnState;
#define TI_RTC_CONN_STATE_IDLE ((TiRtcConnState)0)
#define TI_RTC_CONN_STATE_CONNECTING ((TiRtcConnState)1)
#define TI_RTC_CONN_STATE_CONNECTED ((TiRtcConnState)2)
#define TI_RTC_CONN_STATE_DISCONNECTED ((TiRtcConnState)3)

typedef struct TiRtcInitOptions {
  const char* app_id;
  const char* endpoint;
  const char* cache_root_dir;
  uint8_t console_log_enabled;
} TiRtcInitOptions;
#define TI_RTC_INIT_OPTIONS_INITIALIZER {NULL, NULL, NULL, 0u}

typedef struct TiRtcConnServiceStartOptions {
  const char* device_id;
  const char* device_secret_key;
  const char* client_id;
  uint32_t max_connections;
} TiRtcConnServiceStartOptions;
#define TI_RTC_CONN_SERVICE_START_OPTIONS_INITIALIZER {NULL, NULL, NULL, 0u}

typedef struct TiRtcConnConnectOptions {
  const char* remote_id;
  const char* token;
} TiRtcConnConnectOptions;
#define TI_RTC_CONN_CONNECT_OPTIONS_INITIALIZER {NULL, NULL}

typedef struct TiRtcConnCreateOptions {
  const TiRtcConnCallbacks* callbacks;
  void* user_data;
} TiRtcConnCreateOptions;
#define TI_RTC_CONN_CREATE_OPTIONS_INITIALIZER {NULL, NULL}

typedef struct TiRtcStreamMessage {
  uint32_t timestamp_ms;
  const uint8_t* data;
  uint64_t data_size;
} TiRtcStreamMessage;

typedef struct TiRtcConnCommand {
  uint32_t command;
  const uint8_t* data;
  uint64_t data_size;
} TiRtcConnCommand;

typedef struct TiRtcConnMetricsSnapshot {
  uint8_t has_connect_start;
  uint8_t has_connected;
  uint64_t connect_start_monotonic_ms;
  uint64_t connected_monotonic_ms;
} TiRtcConnMetricsSnapshot;

typedef struct TiRtcOutputStartupMetrics {
  uint8_t has_connect_start;
  uint8_t has_connected;
  uint8_t has_first_packet;
  uint8_t has_first_output;
  uint64_t connect_start_monotonic_ms;
  uint64_t connected_monotonic_ms;
  uint64_t first_packet_monotonic_ms;
  uint64_t first_output_monotonic_ms;
  int64_t first_packet_after_connected_ms;
  int64_t first_output_after_connected_ms;
  int64_t time_to_first_output_ms;
} TiRtcOutputStartupMetrics;

typedef struct TiRtcOutputStutterMetrics {
  uint32_t stutter_threshold_ms;
  uint32_t stutter_count;
  uint64_t output_duration_ms;
  uint64_t stutter_total_ms;
  uint64_t stutter_peak_ms;
  uint64_t stutter_average_ms;
  double stutter_rate;
} TiRtcOutputStutterMetrics;

typedef struct TiRtcAudioOutputMetricsSnapshot {
  TiRtcOutputStartupMetrics startup;
  TiRtcOutputStutterMetrics stutter;
  int64_t estimated_output_latency_ms;
  TiMediaCodec audio_codec;
  uint32_t audio_sample_rate_hz;
  uint32_t audio_channels;
  double audio_input_bitrate_kbps;
  double audio_input_packet_rate;
  double audio_render_callback_rate;
  uint32_t stats_refresh_interval_ms;
  uint64_t stats_updated_at_ms;
} TiRtcAudioOutputMetricsSnapshot;

typedef struct TiRtcVideoOutputMetricsSnapshot {
  TiRtcOutputStartupMetrics startup;
  TiRtcOutputStutterMetrics stutter;
  int64_t estimated_output_latency_ms;
  TiMediaCodec video_codec;
  uint32_t video_width;
  uint32_t video_height;
  TiVideoDecoderBackend decoder_backend;
  double video_input_bitrate_kbps;
  double video_input_fps;
  double video_decoded_fps;
  double video_render_fps;
  uint32_t stats_refresh_interval_ms;
  uint64_t stats_updated_at_ms;
} TiRtcVideoOutputMetricsSnapshot;

typedef void(TI_CALL* TiRtcConnServiceOnStartedFn)(TiRtcConnService* service, void* user_data);
typedef void(TI_CALL* TiRtcConnServiceOnStoppedFn)(TiRtcConnService* service, void* user_data);
/* Transfers one owned connection handle to the callback. Stopping the service does not destroy it.
 */
typedef void(TI_CALL* TiRtcConnServiceOnConnectedFn)(TiRtcConnService* service,
                                                     TiRtcConn* connection, void* user_data);
typedef void(TI_CALL* TiRtcConnServiceOnErrorFn)(TiRtcConnService* service, TiError error,
                                                 const char* message, void* user_data);
typedef struct TiRtcConnServiceCallbacks {
  TiRtcConnServiceOnStartedFn on_started;
  TiRtcConnServiceOnStoppedFn on_stopped;
  TiRtcConnServiceOnConnectedFn on_connected;
  TiRtcConnServiceOnErrorFn on_error;
  TiCallbackDispatcher dispatcher;
} TiRtcConnServiceCallbacks;
#define TI_RTC_CONN_SERVICE_CALLBACKS_INITIALIZER \
  {NULL, NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiRtcConnOnStateChangedFn)(TiRtcConn* connection, TiRtcConnState state,
                                                 TiError error, void* user_data);
typedef void(TI_CALL* TiRtcConnOnCommandFn)(TiRtcConn* connection, uint32_t command,
                                            const uint8_t* data, uint64_t data_size,
                                            void* user_data);
typedef void(TI_CALL* TiRtcConnOnStreamMessageFn)(TiRtcConn* connection, uint8_t stream_id,
                                                  uint32_t timestamp_ms, const uint8_t* data,
                                                  uint64_t data_size, void* user_data);
typedef void(TI_CALL* TiRtcConnOnSubscribeAudioFn)(TiRtcConn* connection, uint8_t stream_id,
                                                   void* user_data);
typedef void(TI_CALL* TiRtcConnOnUnsubscribeAudioFn)(TiRtcConn* connection, uint8_t stream_id,
                                                     void* user_data);
typedef void(TI_CALL* TiRtcConnOnSubscribeVideoFn)(TiRtcConn* connection, uint8_t stream_id,
                                                   void* user_data);
typedef void(TI_CALL* TiRtcConnOnUnsubscribeVideoFn)(TiRtcConn* connection, uint8_t stream_id,
                                                     void* user_data);
struct TiRtcConnCallbacks {
  TiRtcConnOnStateChangedFn on_state_changed;
  TiRtcConnOnCommandFn on_command;
  TiRtcConnOnStreamMessageFn on_stream_message;
  TiRtcConnOnSubscribeAudioFn on_subscribe_audio;
  TiRtcConnOnUnsubscribeAudioFn on_unsubscribe_audio;
  TiRtcConnOnSubscribeVideoFn on_subscribe_video;
  TiRtcConnOnUnsubscribeVideoFn on_unsubscribe_video;
  TiCallbackDispatcher dispatcher;
};
#define TI_RTC_CONN_CALLBACKS_INITIALIZER \
  {NULL, NULL, NULL, NULL, NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

TI_API TiError TI_CALL tirtc_init(const TiRtcInitOptions* options);
TI_API TiError TI_CALL tirtc_uninit(void);
TI_API TiError TI_CALL tirtc_set_connect_link_mode(TiRtcConnectLinkMode mode);

TI_API TiError TI_CALL tirtc_conn_service_start(const TiRtcConnServiceStartOptions* options,
                                                const TiRtcConnServiceCallbacks* callbacks,
                                                void* user_data, TiRtcConnService** out_service);
TI_API TiError TI_CALL tirtc_conn_service_stop(TiRtcConnService* service);
TI_API TiError TI_CALL tirtc_conn_service_destroy(TiRtcConnService* service);

TI_API TiError TI_CALL tirtc_conn_create(const TiRtcConnCreateOptions* options,
                                         TiRtcConn** out_connection);
TI_API TiError TI_CALL tirtc_conn_set_callbacks(TiRtcConn* connection,
                                                const TiRtcConnCallbacks* callbacks,
                                                void* user_data);
TI_API TiError TI_CALL tirtc_conn_connect(TiRtcConn* connection,
                                          const TiRtcConnConnectOptions* options);
TI_API TiError TI_CALL tirtc_conn_disconnect(TiRtcConn* connection);
TI_API TiError TI_CALL tirtc_conn_send_command(TiRtcConn* connection,
                                               const TiRtcConnCommand* command);
TI_API TiError TI_CALL tirtc_conn_send_stream_message(TiRtcConn* connection, uint8_t stream_id,
                                                      const TiRtcStreamMessage* message);
TI_API TiError TI_CALL tirtc_conn_destroy(TiRtcConn* connection);
TI_API TiError TI_CALL tirtc_conn_subscribe_audio(TiRtcConn* connection, uint8_t stream_id);
TI_API TiError TI_CALL tirtc_conn_unsubscribe_audio(TiRtcConn* connection, uint8_t stream_id);
TI_API TiError TI_CALL tirtc_conn_subscribe_video(TiRtcConn* connection, uint8_t stream_id);
TI_API TiError TI_CALL tirtc_conn_unsubscribe_video(TiRtcConn* connection, uint8_t stream_id);
TI_API TiError TI_CALL tirtc_conn_request_video_key_frame(TiRtcConn* connection, uint8_t stream_id);
/*
 * Starts a recording task owned by the caller. A successful task must be stopped before checked
 * destroy. stop is blocking and repeatable; it returns the cached terminal result and borrows the
 * returned file_path until destroy succeeds. The published MP4 remains owned by the caller.
 */
TI_API TiError TI_CALL tirtc_conn_start_recording(TiRtcConn* connection,
                                                  const TiRtcStartRecordingOptions* options,
                                                  TiRtcRecordingTask** out_task);
TI_API TiError TI_CALL tirtc_recording_task_stop(TiRtcRecordingTask* task, TiRtcMp4File* out_file);
TI_API TiError TI_CALL tirtc_recording_task_destroy(TiRtcRecordingTask* task);

/* Raw inputs may be attached and detached while running. */
TI_API TiError TI_CALL tirtc_audio_input_attach(TiAudioInput* input, TiRtcConn* connection,
                                                uint8_t stream_id);
TI_API TiError TI_CALL tirtc_audio_input_detach(TiAudioInput* input, TiRtcConn* connection);
/* Encoded input bindings may only be changed while the input is stopped. */
TI_API TiError TI_CALL tirtc_encoded_audio_input_attach(TiEncodedAudioInput* input,
                                                        TiRtcConn* connection, uint8_t stream_id);
TI_API TiError TI_CALL tirtc_encoded_audio_input_detach(TiEncodedAudioInput* input,
                                                        TiRtcConn* connection);
/* Raw device or submitted-frame inputs may be attached and detached while running. */
TI_API TiError TI_CALL tirtc_video_input_attach(TiVideoInput* input, TiRtcConn* connection,
                                                uint8_t stream_id);
TI_API TiError TI_CALL tirtc_video_input_detach(TiVideoInput* input, TiRtcConn* connection);
TI_API TiError TI_CALL tirtc_encoded_video_input_attach(TiEncodedVideoInput* input,
                                                        TiRtcConn* connection, uint8_t stream_id);
TI_API TiError TI_CALL tirtc_encoded_video_input_detach(TiEncodedVideoInput* input,
                                                        TiRtcConn* connection);

TI_API TiError TI_CALL tirtc_audio_output_attach(TiAudioOutput* output, TiRtcConn* connection,
                                                 uint8_t stream_id);
TI_API TiError TI_CALL tirtc_audio_output_detach(TiAudioOutput* output);
TI_API TiError TI_CALL tirtc_video_output_attach(TiVideoOutput* output, TiRtcConn* connection,
                                                 uint8_t stream_id);
TI_API TiError TI_CALL tirtc_video_output_detach(TiVideoOutput* output);
TI_API TiError TI_CALL tirtc_encoded_audio_output_attach(TiEncodedAudioOutput* output,
                                                         TiRtcConn* connection, uint8_t stream_id);
TI_API TiError TI_CALL tirtc_encoded_audio_output_detach(TiEncodedAudioOutput* output);
TI_API TiError TI_CALL tirtc_encoded_video_output_attach(TiEncodedVideoOutput* output,
                                                         TiRtcConn* connection, uint8_t stream_id);
TI_API TiError TI_CALL tirtc_encoded_video_output_detach(TiEncodedVideoOutput* output);

TI_API TiError TI_CALL tirtc_conn_get_metrics_snapshot(TiRtcConn* connection,
                                                       TiRtcConnMetricsSnapshot* out_snapshot);
TI_API TiError TI_CALL tirtc_audio_output_get_metrics_snapshot(
    TiAudioOutput* output, TiRtcAudioOutputMetricsSnapshot* out_snapshot);
TI_API TiError TI_CALL tirtc_audio_output_reset_metrics_session(TiAudioOutput* output);
TI_API TiError TI_CALL tirtc_video_output_get_metrics_snapshot(
    TiVideoOutput* output, TiRtcVideoOutputMetricsSnapshot* out_snapshot);
TI_API TiError TI_CALL tirtc_video_output_reset_metrics_session(TiVideoOutput* output);

#ifdef __cplusplus
}
#endif

#endif  // TI_RTC_H_
