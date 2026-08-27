#ifndef TI_MEDIA_OHOS_H_
#define TI_MEDIA_OHOS_H_

#include "ti/media.h"

#if defined(__OHOS__)
typedef struct NativeWindow OHNativeWindow;

#ifdef __cplusplus
extern "C" {
#endif

TI_API TiError TI_CALL ti_ohos_audio_input_use_default_device(TiAudioInput* input);
TI_API TiError TI_CALL ti_ohos_audio_output_use_default_device(TiAudioOutput* output);
TI_API TiError TI_CALL ti_ohos_video_input_use_system_camera(TiVideoInput* input,
                                                             TiCameraFacing facing);
TI_API TiError TI_CALL ti_ohos_video_output_set_native_window(TiVideoOutput* output,
                                                              OHNativeWindow* native_window);
TI_API TiError TI_CALL ti_ohos_video_output_clear_target(TiVideoOutput* output);

#ifdef __cplusplus
}
#endif
#endif

#endif  // TI_MEDIA_OHOS_H_
