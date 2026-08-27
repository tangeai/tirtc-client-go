#ifndef TI_MEDIA_APPLE_H_
#define TI_MEDIA_APPLE_H_

#include "ti/media.h"

#if defined(__APPLE__)
#include <CoreVideo/CoreVideo.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef void(TI_CALL* TiAppleVideoOutputOnPixelBufferFn)(TiVideoOutput* output,
                                                         CVPixelBufferRef pixel_buffer,
                                                         void* user_data);

TI_API TiError TI_CALL ti_apple_audio_input_use_default_device(TiAudioInput* input);
TI_API TiError TI_CALL ti_apple_audio_output_use_default_device(TiAudioOutput* output);
TI_API TiError TI_CALL ti_apple_video_input_use_system_camera(TiVideoInput* input,
                                                              TiCameraFacing facing);
TI_API TiError TI_CALL ti_apple_video_output_set_view_target(TiVideoOutput* output, void* view);
TI_API TiError TI_CALL ti_apple_video_output_set_pixel_buffer_target(
    TiVideoOutput* output, TiAppleVideoOutputOnPixelBufferFn on_pixel_buffer, void* user_data,
    const TiCallbackDispatcher* dispatcher);
TI_API TiError TI_CALL ti_apple_video_output_clear_target(TiVideoOutput* output);

#ifdef __cplusplus
}
#endif
#endif

#endif  // TI_MEDIA_APPLE_H_
