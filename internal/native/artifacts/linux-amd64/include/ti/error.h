#ifndef TI_ERROR_H_
#define TI_ERROR_H_

#include <stdint.h>

#if defined(_WIN32)
#if defined(TI_RUNTIME_IMPLEMENTATION)
#define TI_API __declspec(dllexport)
#else
#define TI_API __declspec(dllimport)
#endif
#define TI_CALL __cdecl
#else
#define TI_API __attribute__((visibility("default")))
#define TI_CALL
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef int32_t TiError;

#define TI_ERROR_OK ((TiError)0)
#define TI_ERROR_INVALID_ARGUMENT ((TiError)6000)
#define TI_ERROR_NOT_INITIALIZED ((TiError)6001)
#define TI_ERROR_AUTHENTICATION_FAILED ((TiError)6008)
#define TI_ERROR_TIMEOUT ((TiError)6009)
#define TI_ERROR_REMOTE_CLOSED ((TiError)6012)
#define TI_ERROR_TOKEN_EXPIRED ((TiError)6014)
#define TI_ERROR_ALREADY_INITIALIZED ((TiError)6022)
#define TI_ERROR_PERMISSION_DENIED ((TiError)6024)
#define TI_ERROR_IN_USE ((TiError)6026)
#define TI_ERROR_NOT_STARTED ((TiError)6027)
#define TI_ERROR_NOT_CONNECTED ((TiError)6028)
#define TI_ERROR_NOT_BOUND ((TiError)6029)
#define TI_ERROR_NOT_CONFIGURED ((TiError)6030)
#define TI_ERROR_APP_ID_REQUIRED ((TiError)6032)
#define TI_ERROR_RESOURCE_EXHAUSTED ((TiError)6043)
#define TI_ERROR_FILE_OPEN_FAILED ((TiError)6044)
#define TI_ERROR_FILE_READ_FAILED ((TiError)6045)
#define TI_ERROR_FILE_WRITE_FAILED ((TiError)6046)
#define TI_ERROR_LOG_EXPORT_FAILED ((TiError)6048)
#define TI_ERROR_LOG_UPLOAD_FAILED ((TiError)6049)
#define TI_ERROR_LOG_ARCHIVE_WRITE_FAILED ((TiError)6073)
#define TI_ERROR_UNSUPPORTED ((TiError)6107)
#define TI_ERROR_DEVICE_NOT_FOUND ((TiError)6110)
#define TI_ERROR_DEVICE_ACCESS_DENIED ((TiError)6111)
#define TI_ERROR_DEVICE_BUSY ((TiError)6112)
#define TI_ERROR_UNSUPPORTED_FORMAT ((TiError)6113)
#define TI_ERROR_IO_FAILED ((TiError)6114)
#define TI_ERROR_CANCELLED ((TiError)6115)
#define TI_ERROR_RANGE_TOO_LARGE ((TiError)6117)
#define TI_ERROR_NO_FRAME ((TiError)6118)
#define TI_ERROR_NO_RECORDABLE_MEDIA ((TiError)6119)
#define TI_ERROR_RECORDING_OVERRUN ((TiError)6120)
#define TI_ERROR_OUTPUT_PATH_UNAVAILABLE ((TiError)6121)
#define TI_ERROR_LOG_EXPORT_STAGE_FAILED ((TiError)6125)
#define TI_ERROR_LOG_RAW_DUMP_SNAPSHOT_FAILED ((TiError)6126)
#define TI_ERROR_LOG_UPLOAD_RESULT_INVALID ((TiError)6127)
#define TI_ERROR_LOG_UPLOAD_LOG_ID_TOO_LARGE ((TiError)6128)
#define TI_ERROR_LOG_UPLOAD_CLEANUP_FAILED ((TiError)6129)
#define TI_ERROR_MEDIA_STREAM_TIMEOUT ((TiError)6130)
#define TI_ERROR_LOG_UPLOAD_SERVICE_DISCOVERY_FAILED ((TiError)6131)
#define TI_ERROR_LOG_UPLOAD_CREDENTIAL_FAILED ((TiError)6132)
#define TI_ERROR_LOG_UPLOAD_OBJECT_FAILED ((TiError)6133)

TI_API const char* TI_CALL ti_error_to_string(TiError error);

#ifdef __cplusplus
}
#endif

#endif  // TI_ERROR_H_
