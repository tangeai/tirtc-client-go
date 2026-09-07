#ifndef TI_MEDIA_H_
#define TI_MEDIA_H_

#include <stddef.h>
#include <stdint.h>

#include "ti/error.h"
#include "ti/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif

#define TI_VIDEO_MAX_PLANES 4u

typedef struct TiAudioInput TiAudioInput;
typedef struct TiEncodedAudioInput TiEncodedAudioInput;
typedef struct TiVideoInput TiVideoInput;
typedef struct TiEncodedVideoInput TiEncodedVideoInput;
typedef struct TiAudioOutput TiAudioOutput;
typedef struct TiVideoOutput TiVideoOutput;
typedef struct TiEncodedAudioOutput TiEncodedAudioOutput;
typedef struct TiEncodedVideoOutput TiEncodedVideoOutput;

typedef struct TiVideoSnapshotFile {
  const char* file_path;
} TiVideoSnapshotFile;

#define TI_VIDEO_SNAPSHOT_FILE_INITIALIZER {NULL}

/*
 * Deletes a TiRTC- or Ti Cloud Storage-generated temporary MP4 or JPEG under the active Runtime cache
 * root. Either the RTC or Ti Cloud Storage product lease must be initialized.
 * The operation is idempotent. Paths outside the TiRTC media cache, directories, and symbolic
 * links are rejected; this function is not a general-purpose filesystem delete primitive.
 */
TI_API TiError TI_CALL ti_local_media_file_delete(const char* file_path);

typedef uint32_t TiMediaCodec;
#define TI_MEDIA_CODEC_NONE ((TiMediaCodec)0)
#define TI_MEDIA_CODEC_AUDIO_G711A ((TiMediaCodec)1)
#define TI_MEDIA_CODEC_AUDIO_AAC ((TiMediaCodec)2)
#define TI_MEDIA_CODEC_AUDIO_PCM ((TiMediaCodec)3)
#define TI_MEDIA_CODEC_AUDIO_OPUS ((TiMediaCodec)4)
#define TI_MEDIA_CODEC_AUDIO_AMR ((TiMediaCodec)5)
#define TI_MEDIA_CODEC_VIDEO_H264 ((TiMediaCodec)65)
#define TI_MEDIA_CODEC_VIDEO_H265 ((TiMediaCodec)66)
#define TI_MEDIA_CODEC_VIDEO_MJPEG ((TiMediaCodec)67)

typedef uint32_t TiAudioBitstreamFormat;
#define TI_AUDIO_BITSTREAM_FORMAT_NONE ((TiAudioBitstreamFormat)0)
#define TI_AUDIO_BITSTREAM_FORMAT_G711A_PACKET ((TiAudioBitstreamFormat)1)
#define TI_AUDIO_BITSTREAM_FORMAT_AAC_ADTS ((TiAudioBitstreamFormat)2)
#define TI_AUDIO_BITSTREAM_FORMAT_AAC_RAW_ACCESS_UNIT ((TiAudioBitstreamFormat)3)
#define TI_AUDIO_BITSTREAM_FORMAT_PCM_S16LE_INTERLEAVED ((TiAudioBitstreamFormat)4)
#define TI_AUDIO_BITSTREAM_FORMAT_OPUS_PACKET ((TiAudioBitstreamFormat)5)
#define TI_AUDIO_BITSTREAM_FORMAT_AMR_NB_FRAME ((TiAudioBitstreamFormat)6)

typedef uint32_t TiVideoBitstreamFormat;
#define TI_VIDEO_BITSTREAM_FORMAT_NONE ((TiVideoBitstreamFormat)0)
#define TI_VIDEO_BITSTREAM_FORMAT_H264_ANNEXB ((TiVideoBitstreamFormat)1)
#define TI_VIDEO_BITSTREAM_FORMAT_H265_ANNEXB ((TiVideoBitstreamFormat)2)
#define TI_VIDEO_BITSTREAM_FORMAT_MJPEG_JFIF ((TiVideoBitstreamFormat)3)

typedef uint32_t TiAudioSampleFormat;
#define TI_AUDIO_SAMPLE_FORMAT_NONE ((TiAudioSampleFormat)0)
#define TI_AUDIO_SAMPLE_FORMAT_S16LE_INTERLEAVED ((TiAudioSampleFormat)1)

typedef uint32_t TiVideoPixelFormat;
#define TI_VIDEO_PIXEL_FORMAT_NONE ((TiVideoPixelFormat)0)
#define TI_VIDEO_PIXEL_FORMAT_I420 ((TiVideoPixelFormat)1)
#define TI_VIDEO_PIXEL_FORMAT_NV12 ((TiVideoPixelFormat)2)
#define TI_VIDEO_PIXEL_FORMAT_RGBA8888 ((TiVideoPixelFormat)3)

typedef uint32_t TiCameraFacing;
/** Selects the platform default camera; platforms without a native default use the front camera. */
#define TI_CAMERA_FACING_UNSPECIFIED ((TiCameraFacing)0)
#define TI_CAMERA_FACING_FRONT ((TiCameraFacing)1)
#define TI_CAMERA_FACING_BACK ((TiCameraFacing)2)

typedef uint32_t TiInputState;
#define TI_INPUT_STATE_IDLE ((TiInputState)0)
#define TI_INPUT_STATE_RUNNING ((TiInputState)1)
#define TI_INPUT_STATE_STOPPED ((TiInputState)2)
#define TI_INPUT_STATE_FAILED ((TiInputState)3)

typedef uint32_t TiOutputState;
#define TI_OUTPUT_STATE_IDLE ((TiOutputState)0)
#define TI_OUTPUT_STATE_BUFFERING ((TiOutputState)1)
#define TI_OUTPUT_STATE_DELIVERING ((TiOutputState)2)
#define TI_OUTPUT_STATE_FAILED ((TiOutputState)3)
#define TI_OUTPUT_STATE_PAUSED ((TiOutputState)4)
#define TI_OUTPUT_STATE_COMPLETED ((TiOutputState)5)

typedef uint32_t TiOutputBufferStrategy;
#define TI_OUTPUT_BUFFER_STRATEGY_AUTOMATIC ((TiOutputBufferStrategy)0)
#define TI_OUTPUT_BUFFER_STRATEGY_NO_BUFFER ((TiOutputBufferStrategy)1)

typedef uint32_t TiAudioAecMode;
#define TI_AUDIO_AEC_MODE_DISABLED ((TiAudioAecMode)0)
#define TI_AUDIO_AEC_MODE_ENABLED ((TiAudioAecMode)1)

typedef uint32_t TiAudioAgcLevel;
#define TI_AUDIO_AGC_LEVEL_DISABLED ((TiAudioAgcLevel)0)
#define TI_AUDIO_AGC_LEVEL_LOW ((TiAudioAgcLevel)1)
#define TI_AUDIO_AGC_LEVEL_MEDIUM ((TiAudioAgcLevel)2)
#define TI_AUDIO_AGC_LEVEL_HIGH ((TiAudioAgcLevel)3)

typedef uint32_t TiAudioAnsLevel;
#define TI_AUDIO_ANS_LEVEL_DISABLED ((TiAudioAnsLevel)0)
#define TI_AUDIO_ANS_LEVEL_LOW ((TiAudioAnsLevel)1)
#define TI_AUDIO_ANS_LEVEL_MEDIUM ((TiAudioAnsLevel)2)
#define TI_AUDIO_ANS_LEVEL_HIGH ((TiAudioAnsLevel)3)

typedef uint32_t TiVideoDecoderPreference;
#define TI_VIDEO_DECODER_PREFERENCE_AUTO ((TiVideoDecoderPreference)0)
#define TI_VIDEO_DECODER_PREFERENCE_SOFTWARE ((TiVideoDecoderPreference)1)
#define TI_VIDEO_DECODER_PREFERENCE_HARDWARE ((TiVideoDecoderPreference)2)

typedef uint32_t TiVideoDecoderBackend;
#define TI_VIDEO_DECODER_BACKEND_UNKNOWN ((TiVideoDecoderBackend)0)
#define TI_VIDEO_DECODER_BACKEND_SOFTWARE ((TiVideoDecoderBackend)1)
#define TI_VIDEO_DECODER_BACKEND_HARDWARE ((TiVideoDecoderBackend)2)

typedef struct TiAudioFormat {
  TiAudioSampleFormat sample_format;
  uint32_t sample_rate_hz;
  uint32_t channels;
} TiAudioFormat;

typedef struct TiAudioFrame TiAudioFrame;
typedef struct TiEncodedAudioFrame TiEncodedAudioFrame;
typedef struct TiVideoFrame TiVideoFrame;
typedef struct TiEncodedVideoFrame TiEncodedVideoFrame;

typedef struct TiVideoPlaneInfo {
  const uint8_t* data;
  uint64_t data_size;
  uint32_t stride_bytes;
} TiVideoPlaneInfo;

typedef struct TiAudioFrameInfo {
  int64_t pts_us;
  int64_t source_time_utc_us;
  const uint8_t* data;
  uint64_t data_size;
  TiAudioFormat format;
  uint32_t samples_per_channel;
  uint8_t has_source_time;
  uint8_t discontinuity;
} TiAudioFrameInfo;

#define TI_AUDIO_FRAME_INFO_INITIALIZER \
  {0, 0, NULL, 0, {TI_AUDIO_SAMPLE_FORMAT_NONE, 0u, 0u}, 0u, 0u, 0u}

typedef struct TiEncodedAudioFrameInfo {
  int64_t pts_us;
  int64_t source_time_utc_us;
  const uint8_t* data;
  uint64_t data_size;
  const uint8_t* codec_config;
  uint64_t codec_config_size;
  TiMediaCodec codec;
  TiAudioBitstreamFormat bitstream_format;
  uint32_t sample_rate_hz;
  uint32_t channels;
  uint8_t has_source_time;
  uint8_t discontinuity;
} TiEncodedAudioFrameInfo;

#define TI_ENCODED_AUDIO_FRAME_INFO_INITIALIZER \
  {0, 0, NULL, 0, NULL, 0, TI_MEDIA_CODEC_NONE, TI_AUDIO_BITSTREAM_FORMAT_NONE, 0u, 0u, 0u, 0u}

typedef struct TiVideoFrameInfo {
  int64_t pts_us;
  int64_t source_time_utc_us;
  TiVideoPlaneInfo planes[TI_VIDEO_MAX_PLANES];
  uint32_t width;
  uint32_t height;
  TiVideoPixelFormat pixel_format;
  uint32_t plane_count;
  uint8_t has_source_time;
  uint8_t discontinuity;
} TiVideoFrameInfo;

#define TI_VIDEO_FRAME_INFO_INITIALIZER                                  \
  {0,  0,  {{NULL, 0, 0u}, {NULL, 0, 0u}, {NULL, 0, 0u}, {NULL, 0, 0u}}, \
   0u, 0u, TI_VIDEO_PIXEL_FORMAT_NONE,                                   \
   0u, 0u, 0u}

typedef struct TiEncodedVideoFrameInfo {
  int64_t pts_us;
  int64_t source_time_utc_us;
  const uint8_t* data;
  uint64_t data_size;
  const uint8_t* codec_config;
  uint64_t codec_config_size;
  TiMediaCodec codec;
  TiVideoBitstreamFormat bitstream_format;
  uint32_t width;
  uint32_t height;
  uint8_t key_frame;
  uint8_t has_source_time;
  uint8_t discontinuity;
} TiEncodedVideoFrameInfo;

#define TI_ENCODED_VIDEO_FRAME_INFO_INITIALIZER \
  {0, 0, NULL, 0, NULL, 0, TI_MEDIA_CODEC_NONE, TI_VIDEO_BITSTREAM_FORMAT_NONE, 0u, 0u, 0u, 0u, 0u}

typedef struct TiAudioOutputDebugSnapshot {
  TiMediaCodec codec;
  TiAudioBitstreamFormat bitstream_format;
  uint32_t sample_rate_hz;
  uint32_t channels;
} TiAudioOutputDebugSnapshot;

typedef struct TiVideoOutputDebugSnapshot {
  TiMediaCodec codec;
  TiVideoBitstreamFormat bitstream_format;
  uint32_t width;
  uint32_t height;
  TiVideoDecoderPreference requested_decoder_preference;
  TiVideoDecoderBackend resolved_decoder_backend;
} TiVideoOutputDebugSnapshot;

typedef struct TiAudioInputOptions {
  TiMediaCodec codec;
  TiAudioBitstreamFormat bitstream_format;
  uint32_t sample_rate_hz;
  uint32_t channels;
  TiAudioAecMode aec_mode;
  TiAudioAgcLevel agc_level;
  TiAudioAnsLevel ans_level;
} TiAudioInputOptions;

#define TI_AUDIO_INPUT_OPTIONS_INITIALIZER \
  {                                        \
      TI_MEDIA_CODEC_NONE,                 \
      TI_AUDIO_BITSTREAM_FORMAT_NONE,      \
      0u,                                  \
      0u,                                  \
      TI_AUDIO_AEC_MODE_DISABLED,          \
      TI_AUDIO_AGC_LEVEL_DISABLED,         \
      TI_AUDIO_ANS_LEVEL_DISABLED}

typedef struct TiEncodedAudioInputOptions {
  TiMediaCodec codec;
  TiAudioBitstreamFormat bitstream_format;
  uint32_t sample_rate_hz;
  uint32_t channels;
} TiEncodedAudioInputOptions;

#define TI_ENCODED_AUDIO_INPUT_OPTIONS_INITIALIZER \
  {TI_MEDIA_CODEC_NONE, TI_AUDIO_BITSTREAM_FORMAT_NONE, 0u, 0u}

typedef struct TiVideoInputOptions {
  TiMediaCodec codec;
  TiVideoBitstreamFormat bitstream_format;
  TiVideoPixelFormat input_pixel_format;
  uint32_t width;
  uint32_t height;
  uint32_t fps;
  uint32_t bitrate_kbps;
} TiVideoInputOptions;

#define TI_VIDEO_INPUT_OPTIONS_INITIALIZER \
  {TI_MEDIA_CODEC_NONE, TI_VIDEO_BITSTREAM_FORMAT_NONE, TI_VIDEO_PIXEL_FORMAT_NONE, 0u, 0u, 0u, 0u}

typedef struct TiEncodedVideoInputOptions {
  TiMediaCodec codec;
  TiVideoBitstreamFormat bitstream_format;
} TiEncodedVideoInputOptions;

#define TI_ENCODED_VIDEO_INPUT_OPTIONS_INITIALIZER \
  {TI_MEDIA_CODEC_NONE, TI_VIDEO_BITSTREAM_FORMAT_NONE}

typedef struct TiAudioOutputOptions {
  TiAudioAgcLevel agc_level;
  TiAudioAnsLevel ans_level;
} TiAudioOutputOptions;

#define TI_AUDIO_OUTPUT_OPTIONS_INITIALIZER \
  {TI_AUDIO_AGC_LEVEL_DISABLED, TI_AUDIO_ANS_LEVEL_DISABLED}

typedef struct TiVideoOutputOptions {
  TiVideoDecoderPreference decoder_preference;
} TiVideoOutputOptions;

#define TI_VIDEO_OUTPUT_OPTIONS_INITIALIZER {TI_VIDEO_DECODER_PREFERENCE_AUTO}

typedef struct TiOutputBufferOptions {
  TiOutputBufferStrategy strategy;
  uint8_t has_max_buffer_watermark_ms;
  int32_t max_buffer_watermark_ms;
} TiOutputBufferOptions;

#define TI_OUTPUT_BUFFER_OPTIONS_INITIALIZER {TI_OUTPUT_BUFFER_STRATEGY_AUTOMATIC, 0u, 0}

typedef void(TI_CALL* TiAudioInputOnStateChangedFn)(TiAudioInput* input, TiInputState state,
                                                    void* user_data);
typedef void(TI_CALL* TiAudioInputOnErrorFn)(TiAudioInput* input, TiError error,
                                             const char* message, void* user_data);
typedef struct TiAudioInputCallbacks {
  TiAudioInputOnStateChangedFn on_state_changed;
  TiAudioInputOnErrorFn on_error;
  TiCallbackDispatcher dispatcher;
} TiAudioInputCallbacks;
#define TI_AUDIO_INPUT_CALLBACKS_INITIALIZER {NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiEncodedAudioInputOnStateChangedFn)(TiEncodedAudioInput* input,
                                                           TiInputState state, void* user_data);
typedef void(TI_CALL* TiEncodedAudioInputOnErrorFn)(TiEncodedAudioInput* input, TiError error,
                                                    const char* message, void* user_data);
typedef struct TiEncodedAudioInputCallbacks {
  TiEncodedAudioInputOnStateChangedFn on_state_changed;
  TiEncodedAudioInputOnErrorFn on_error;
  TiCallbackDispatcher dispatcher;
} TiEncodedAudioInputCallbacks;
#define TI_ENCODED_AUDIO_INPUT_CALLBACKS_INITIALIZER \
  {NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiVideoInputOnStateChangedFn)(TiVideoInput* input, TiInputState state,
                                                    void* user_data);
typedef void(TI_CALL* TiVideoInputOnOutputSizeChangedFn)(TiVideoInput* input, uint32_t width,
                                                         uint32_t height, void* user_data);
typedef void(TI_CALL* TiVideoInputOnErrorFn)(TiVideoInput* input, TiError error,
                                             const char* message, void* user_data);
typedef struct TiVideoInputCallbacks {
  TiVideoInputOnStateChangedFn on_state_changed;
  TiVideoInputOnOutputSizeChangedFn on_output_size_changed;
  TiVideoInputOnErrorFn on_error;
  TiCallbackDispatcher dispatcher;
} TiVideoInputCallbacks;
#define TI_VIDEO_INPUT_CALLBACKS_INITIALIZER {NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiEncodedVideoInputOnStateChangedFn)(TiEncodedVideoInput* input,
                                                           TiInputState state, void* user_data);
typedef void(TI_CALL* TiEncodedVideoInputOnOutputSizeChangedFn)(TiEncodedVideoInput* input,
                                                                uint32_t width, uint32_t height,
                                                                void* user_data);
typedef void(TI_CALL* TiEncodedVideoInputOnErrorFn)(TiEncodedVideoInput* input, TiError error,
                                                    const char* message, void* user_data);
typedef void(TI_CALL* TiEncodedVideoInputOnKeyFrameRequestedFn)(TiEncodedVideoInput* input,
                                                                void* user_data);
typedef struct TiEncodedVideoInputCallbacks {
  TiEncodedVideoInputOnStateChangedFn on_state_changed;
  TiEncodedVideoInputOnOutputSizeChangedFn on_output_size_changed;
  TiEncodedVideoInputOnErrorFn on_error;
  TiEncodedVideoInputOnKeyFrameRequestedFn on_key_frame_requested;
  TiCallbackDispatcher dispatcher;
} TiEncodedVideoInputCallbacks;
#define TI_ENCODED_VIDEO_INPUT_CALLBACKS_INITIALIZER \
  {NULL, NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiAudioOutputOnFrameFn)(TiAudioOutput* output, const TiAudioFrame* frame,
                                              void* user_data);
typedef void(TI_CALL* TiAudioOutputOnStateChangedFn)(TiAudioOutput* output, TiOutputState state,
                                                     void* user_data);
typedef void(TI_CALL* TiAudioOutputOnErrorFn)(TiAudioOutput* output, TiError error,
                                              const char* message, void* user_data);
typedef struct TiAudioOutputCallbacks {
  TiAudioOutputOnFrameFn on_frame;
  TiAudioOutputOnStateChangedFn on_state_changed;
  TiAudioOutputOnErrorFn on_error;
  TiCallbackDispatcher dispatcher;
} TiAudioOutputCallbacks;
#define TI_AUDIO_OUTPUT_CALLBACKS_INITIALIZER {NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiVideoOutputOnFrameFn)(TiVideoOutput* output, const TiVideoFrame* frame,
                                              void* user_data);
typedef void(TI_CALL* TiVideoOutputOnStateChangedFn)(TiVideoOutput* output, TiOutputState state,
                                                     void* user_data);
typedef void(TI_CALL* TiVideoOutputOnOutputSizeChangedFn)(TiVideoOutput* output, uint32_t width,
                                                          uint32_t height, void* user_data);
typedef void(TI_CALL* TiVideoOutputOnErrorFn)(TiVideoOutput* output, TiError error,
                                              const char* message, void* user_data);
typedef struct TiVideoOutputCallbacks {
  TiVideoOutputOnFrameFn on_frame;
  TiVideoOutputOnStateChangedFn on_state_changed;
  TiVideoOutputOnOutputSizeChangedFn on_output_size_changed;
  TiVideoOutputOnErrorFn on_error;
  TiCallbackDispatcher dispatcher;
} TiVideoOutputCallbacks;
#define TI_VIDEO_OUTPUT_CALLBACKS_INITIALIZER \
  {NULL, NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiEncodedAudioOutputOnFrameFn)(TiEncodedAudioOutput* output,
                                                     const TiEncodedAudioFrame* frame,
                                                     void* user_data);
typedef void(TI_CALL* TiEncodedAudioOutputOnStateChangedFn)(TiEncodedAudioOutput* output,
                                                            TiOutputState state, void* user_data);
typedef void(TI_CALL* TiEncodedAudioOutputOnErrorFn)(TiEncodedAudioOutput* output, TiError error,
                                                     const char* message, void* user_data);
typedef struct TiEncodedAudioOutputCallbacks {
  TiEncodedAudioOutputOnFrameFn on_frame;
  TiEncodedAudioOutputOnStateChangedFn on_state_changed;
  TiEncodedAudioOutputOnErrorFn on_error;
  TiCallbackDispatcher dispatcher;
} TiEncodedAudioOutputCallbacks;
#define TI_ENCODED_AUDIO_OUTPUT_CALLBACKS_INITIALIZER \
  {NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

typedef void(TI_CALL* TiEncodedVideoOutputOnFrameFn)(TiEncodedVideoOutput* output,
                                                     const TiEncodedVideoFrame* frame,
                                                     void* user_data);
typedef void(TI_CALL* TiEncodedVideoOutputOnStateChangedFn)(TiEncodedVideoOutput* output,
                                                            TiOutputState state, void* user_data);
typedef void(TI_CALL* TiEncodedVideoOutputOnErrorFn)(TiEncodedVideoOutput* output, TiError error,
                                                     const char* message, void* user_data);
typedef struct TiEncodedVideoOutputCallbacks {
  TiEncodedVideoOutputOnFrameFn on_frame;
  TiEncodedVideoOutputOnStateChangedFn on_state_changed;
  TiEncodedVideoOutputOnErrorFn on_error;
  TiCallbackDispatcher dispatcher;
} TiEncodedVideoOutputCallbacks;
#define TI_ENCODED_VIDEO_OUTPUT_CALLBACKS_INITIALIZER \
  {NULL, NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

/*
 * Frame objects are immutable and thread-safe reference-counted values. A callback lends its
 * frame reference for the duration of that callback. Retain it before returning when another
 * thread or a later operation needs the frame, and release every retained or created reference.
 * Input submission does not consume the caller's reference.
 */
TI_API TiError TI_CALL ti_audio_frame_get_info(const TiAudioFrame* frame,
                                               TiAudioFrameInfo* out_info);
TI_API TiAudioFrame* TI_CALL ti_audio_frame_retain(const TiAudioFrame* frame);
TI_API void TI_CALL ti_audio_frame_release(TiAudioFrame* frame);

TI_API TiError TI_CALL ti_video_frame_create_copy(const TiVideoFrameInfo* info,
                                                  TiVideoFrame** out_frame);
TI_API TiError TI_CALL ti_video_frame_get_info(const TiVideoFrame* frame,
                                               TiVideoFrameInfo* out_info);
TI_API TiVideoFrame* TI_CALL ti_video_frame_retain(const TiVideoFrame* frame);
TI_API void TI_CALL ti_video_frame_release(TiVideoFrame* frame);

TI_API TiError TI_CALL ti_encoded_audio_frame_create_copy(const TiEncodedAudioFrameInfo* info,
                                                          TiEncodedAudioFrame** out_frame);
TI_API TiError TI_CALL ti_encoded_audio_frame_get_info(const TiEncodedAudioFrame* frame,
                                                       TiEncodedAudioFrameInfo* out_info);
TI_API TiEncodedAudioFrame* TI_CALL ti_encoded_audio_frame_retain(const TiEncodedAudioFrame* frame);
TI_API void TI_CALL ti_encoded_audio_frame_release(TiEncodedAudioFrame* frame);

TI_API TiError TI_CALL ti_encoded_video_frame_create_copy(const TiEncodedVideoFrameInfo* info,
                                                          TiEncodedVideoFrame** out_frame);
TI_API TiError TI_CALL ti_encoded_video_frame_get_info(const TiEncodedVideoFrame* frame,
                                                       TiEncodedVideoFrameInfo* out_info);
TI_API TiEncodedVideoFrame* TI_CALL ti_encoded_video_frame_retain(const TiEncodedVideoFrame* frame);
TI_API void TI_CALL ti_encoded_video_frame_release(TiEncodedVideoFrame* frame);

TI_API TiError TI_CALL ti_audio_input_create(TiAudioInput** out_input);
TI_API TiError TI_CALL ti_audio_input_set_options(TiAudioInput* input,
                                                  const TiAudioInputOptions* options);
TI_API TiError TI_CALL ti_audio_input_set_callbacks(TiAudioInput* input,
                                                    const TiAudioInputCallbacks* callbacks,
                                                    void* user_data);
TI_API TiError TI_CALL ti_audio_input_get_state(TiAudioInput* input, TiInputState* out_state);
/* Device capture may be started before an RTC binding is attached. */
TI_API TiError TI_CALL ti_audio_input_start(TiAudioInput* input);
TI_API TiError TI_CALL ti_audio_input_stop(TiAudioInput* input);
TI_API TiError TI_CALL ti_audio_input_destroy(TiAudioInput* input);

TI_API TiError TI_CALL ti_encoded_audio_input_create(TiEncodedAudioInput** out_input);
TI_API TiError TI_CALL ti_encoded_audio_input_set_options(
    TiEncodedAudioInput* input, const TiEncodedAudioInputOptions* options);
TI_API TiError TI_CALL ti_encoded_audio_input_set_callbacks(
    TiEncodedAudioInput* input, const TiEncodedAudioInputCallbacks* callbacks, void* user_data);
TI_API TiError TI_CALL ti_encoded_audio_input_get_state(TiEncodedAudioInput* input,
                                                        TiInputState* out_state);
TI_API TiError TI_CALL ti_encoded_audio_input_start(TiEncodedAudioInput* input);
TI_API TiError TI_CALL ti_encoded_audio_input_stop(TiEncodedAudioInput* input);
TI_API TiError TI_CALL ti_encoded_audio_input_submit_frame(TiEncodedAudioInput* input,
                                                           const TiEncodedAudioFrame* frame);
TI_API TiError TI_CALL ti_encoded_audio_input_destroy(TiEncodedAudioInput* input);

TI_API TiError TI_CALL ti_video_input_create(TiVideoInput** out_input);
TI_API TiError TI_CALL ti_video_input_set_options(TiVideoInput* input,
                                                  const TiVideoInputOptions* options);
TI_API TiError TI_CALL ti_video_input_set_callbacks(TiVideoInput* input,
                                                    const TiVideoInputCallbacks* callbacks,
                                                    void* user_data);
TI_API TiError TI_CALL ti_video_input_get_state(TiVideoInput* input, TiInputState* out_state);
/* Raw capture/preview may be started before an RTC binding is attached. */
TI_API TiError TI_CALL ti_video_input_start(TiVideoInput* input);
TI_API TiError TI_CALL ti_video_input_stop(TiVideoInput* input);
TI_API TiError TI_CALL ti_video_input_submit_frame(TiVideoInput* input, const TiVideoFrame* frame);
TI_API TiError TI_CALL ti_video_input_attach_preview(TiVideoInput* input,
                                                     TiVideoOutput* preview_output);
TI_API TiError TI_CALL ti_video_input_detach_preview(TiVideoInput* input);
TI_API TiError TI_CALL ti_video_input_destroy(TiVideoInput* input);

TI_API TiError TI_CALL ti_encoded_video_input_create(TiEncodedVideoInput** out_input);
TI_API TiError TI_CALL ti_encoded_video_input_set_options(
    TiEncodedVideoInput* input, const TiEncodedVideoInputOptions* options);
TI_API TiError TI_CALL ti_encoded_video_input_set_callbacks(
    TiEncodedVideoInput* input, const TiEncodedVideoInputCallbacks* callbacks, void* user_data);
TI_API TiError TI_CALL ti_encoded_video_input_get_state(TiEncodedVideoInput* input,
                                                        TiInputState* out_state);
TI_API TiError TI_CALL ti_encoded_video_input_start(TiEncodedVideoInput* input);
TI_API TiError TI_CALL ti_encoded_video_input_stop(TiEncodedVideoInput* input);
TI_API TiError TI_CALL ti_encoded_video_input_submit_frame(TiEncodedVideoInput* input,
                                                           const TiEncodedVideoFrame* frame);
TI_API TiError TI_CALL ti_encoded_video_input_attach_preview(TiEncodedVideoInput* input,
                                                             TiVideoOutput* preview_output);
TI_API TiError TI_CALL ti_encoded_video_input_detach_preview(TiEncodedVideoInput* input);
TI_API TiError TI_CALL ti_encoded_video_input_destroy(TiEncodedVideoInput* input);

TI_API TiError TI_CALL ti_audio_output_create(TiAudioOutput** out_output);
TI_API TiError TI_CALL ti_audio_output_set_options(TiAudioOutput* output,
                                                   const TiAudioOutputOptions* options);
TI_API TiError TI_CALL ti_audio_output_set_buffer_options(TiAudioOutput* output,
                                                          const TiOutputBufferOptions* options);
TI_API TiError TI_CALL ti_audio_output_set_callbacks(TiAudioOutput* output,
                                                     const TiAudioOutputCallbacks* callbacks,
                                                     void* user_data);
TI_API TiError TI_CALL ti_audio_output_set_volume(TiAudioOutput* output, uint32_t volume_percent);
/**
 * Selects whether RTC live audio uses adaptive decoded-PCM playout.
 *
 * Adaptive playout is enabled by default. `enabled` accepts only 0 or 1 and
 * must be configured before the output is attached. Disabling selects the
 * bounded direct decoded-PCM path; it does not change the Ti Cloud Storage timeline
 * playback. Calling while attached returns `TI_ERROR_IN_USE`.
 */
TI_API TiError TI_CALL ti_audio_output_set_adaptive_playout_enabled(TiAudioOutput* output,
                                                                    uint8_t enabled);
TI_API TiError TI_CALL ti_audio_output_get_state(TiAudioOutput* output, TiOutputState* out_state);
TI_API TiError TI_CALL ti_audio_output_get_debug_snapshot(TiAudioOutput* output,
                                                          TiAudioOutputDebugSnapshot* out_snapshot);
TI_API TiError TI_CALL ti_audio_output_destroy(TiAudioOutput* output);

TI_API TiError TI_CALL ti_video_output_create(TiVideoOutput** out_output);
TI_API TiError TI_CALL ti_video_output_set_options(TiVideoOutput* output,
                                                   const TiVideoOutputOptions* options);
TI_API TiError TI_CALL ti_video_output_set_buffer_options(TiVideoOutput* output,
                                                          const TiOutputBufferOptions* options);
TI_API TiError TI_CALL ti_video_output_set_callbacks(TiVideoOutput* output,
                                                     const TiVideoOutputCallbacks* callbacks,
                                                     void* user_data);
TI_API TiError TI_CALL ti_video_output_get_state(TiVideoOutput* output, TiOutputState* out_state);
TI_API TiError TI_CALL ti_video_output_get_debug_snapshot(TiVideoOutput* output,
                                                          TiVideoOutputDebugSnapshot* out_snapshot);
/*
 * Synchronously captures the latest successfully rendered frame without waiting for a new frame.
 * On success, out_file points to the generated temporary JPEG. The path remains valid until the
 * next snapshot call on this output or until the output is destroyed; copy it before either event.
 * Delete the JPEG with ti_local_media_file_delete when it is no longer needed.
 */
TI_API TiError TI_CALL ti_video_output_take_snapshot(TiVideoOutput* output,
                                                     TiVideoSnapshotFile* out_file);
TI_API TiError TI_CALL ti_video_output_destroy(TiVideoOutput* output);

TI_API TiError TI_CALL ti_encoded_audio_output_create(TiEncodedAudioOutput** out_output);
TI_API TiError TI_CALL ti_encoded_audio_output_set_callbacks(
    TiEncodedAudioOutput* output, const TiEncodedAudioOutputCallbacks* callbacks, void* user_data);
TI_API TiError TI_CALL ti_encoded_audio_output_get_state(TiEncodedAudioOutput* output,
                                                         TiOutputState* out_state);
TI_API TiError TI_CALL ti_encoded_audio_output_destroy(TiEncodedAudioOutput* output);

TI_API TiError TI_CALL ti_encoded_video_output_create(TiEncodedVideoOutput** out_output);
TI_API TiError TI_CALL ti_encoded_video_output_set_callbacks(
    TiEncodedVideoOutput* output, const TiEncodedVideoOutputCallbacks* callbacks, void* user_data);
TI_API TiError TI_CALL ti_encoded_video_output_get_state(TiEncodedVideoOutput* output,
                                                         TiOutputState* out_state);
TI_API TiError TI_CALL ti_encoded_video_output_destroy(TiEncodedVideoOutput* output);

#ifdef __cplusplus
}
#endif

#endif  // TI_MEDIA_H_
