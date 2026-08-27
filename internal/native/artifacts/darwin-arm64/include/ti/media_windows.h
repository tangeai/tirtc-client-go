#ifndef TI_MEDIA_WINDOWS_H_
#define TI_MEDIA_WINDOWS_H_

#include "ti/media.h"

#if defined(_WIN32)
typedef struct HWND__* TiWindowsHwnd;

#ifdef __cplusplus
extern "C" {
#endif

TI_API TiError TI_CALL ti_windows_audio_input_use_default_device(TiAudioInput* input);
TI_API TiError TI_CALL ti_windows_audio_output_use_default_device(TiAudioOutput* output);
TI_API TiError TI_CALL ti_windows_video_input_use_system_camera(TiVideoInput* input,
                                                                TiCameraFacing facing);
TI_API TiError TI_CALL ti_windows_video_output_set_hwnd(TiVideoOutput* output, TiWindowsHwnd hwnd);
TI_API TiError TI_CALL ti_windows_video_output_clear_target(TiVideoOutput* output);

#ifdef __cplusplus
}
#endif
#endif

#endif  // TI_MEDIA_WINDOWS_H_
