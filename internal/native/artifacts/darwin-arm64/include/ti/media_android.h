#ifndef TI_MEDIA_ANDROID_H_
#define TI_MEDIA_ANDROID_H_

#include "ti/media.h"

#if defined(__ANDROID__)
struct ANativeWindow;

#ifdef __cplusplus
extern "C" {
#endif

TI_API TiError TI_CALL ti_android_audio_input_use_default_device(TiAudioInput* input);
TI_API TiError TI_CALL ti_android_audio_output_use_default_device(TiAudioOutput* output);
TI_API TiError TI_CALL ti_android_video_output_set_native_window(
    TiVideoOutput* output, struct ANativeWindow* native_window);
TI_API TiError TI_CALL ti_android_video_output_clear_target(TiVideoOutput* output);

#ifdef __cplusplus
}
#endif
#endif

#endif  // TI_MEDIA_ANDROID_H_
