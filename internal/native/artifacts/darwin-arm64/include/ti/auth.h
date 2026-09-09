#ifndef TI_AUTH_H_
#define TI_AUTH_H_

#include <stdint.h>

#include "ti/error.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct TiAuthCloudStorageTokenRequest TiAuthCloudStorageTokenRequest;

typedef struct TiAuthToken {
  char* token;
  int64_t expires_at_ms;
} TiAuthToken;

#define TI_AUTH_TOKEN_INITIALIZER {NULL, 0}

typedef struct TiAuthRtcTokenOptions {
  const char* access_key_id;
  const char* access_key_secret;
  const char* device_id;
  const char* subject;
  int64_t ttl_seconds;
} TiAuthRtcTokenOptions;

#define TI_AUTH_RTC_TOKEN_OPTIONS_INITIALIZER {NULL, NULL, NULL, NULL, 0}

typedef struct TiAuthCloudStorageAppTokenOptions {
  const char* app_id;
  const char* access_key_id;
  const char* access_key_secret;
  const char* endpoint;
  const char* device_id;
  int64_t duration_seconds;
} TiAuthCloudStorageAppTokenOptions;

#define TI_AUTH_CLOUD_STORAGE_APP_TOKEN_OPTIONS_INITIALIZER {NULL, NULL, NULL, NULL, NULL, 0}

typedef struct TiAuthCloudStorageDeviceTokenOptions {
  const char* app_id;
  const char* access_key_id;
  const char* access_key_secret;
  const char* endpoint;
  const char* device_id;
  int64_t duration_seconds;
  int32_t recording_retention_days;
} TiAuthCloudStorageDeviceTokenOptions;

#define TI_AUTH_CLOUD_STORAGE_DEVICE_TOKEN_OPTIONS_INITIALIZER {NULL, NULL, NULL, NULL, NULL, 0, 0}

/*
 * Issues one RTC connection token synchronously. The returned token is owned by
 * the caller and must be released with ti_auth_token_release(). This operation
 * does not initialize RTC or Ti Cloud Storage.
 */
TI_API TiError TI_CALL ti_auth_issue_rtc_token(const TiAuthRtcTokenOptions* options,
                                               TiAuthToken* out_token);

/*
 * Request creation copies credentials but performs no network I/O. execute is
 * a single blocking operation and may be cancelled from another thread. Once
 * execute returns, destroy is the I/O and credential-release barrier.
 */
TI_API TiError TI_CALL ti_auth_cloud_storage_app_token_request_create(
    const TiAuthCloudStorageAppTokenOptions* options, TiAuthCloudStorageTokenRequest** out_request);
TI_API TiError TI_CALL ti_auth_cloud_storage_device_token_request_create(
    const TiAuthCloudStorageDeviceTokenOptions* options,
    TiAuthCloudStorageTokenRequest** out_request);
TI_API TiError TI_CALL ti_auth_cloud_storage_token_request_execute(
    TiAuthCloudStorageTokenRequest* request, TiAuthToken* out_token);
TI_API TiError TI_CALL
ti_auth_cloud_storage_token_request_cancel(TiAuthCloudStorageTokenRequest* request);
TI_API TiError TI_CALL
ti_auth_cloud_storage_token_request_destroy(TiAuthCloudStorageTokenRequest* request);

TI_API void TI_CALL ti_auth_token_release(TiAuthToken* token);

#ifdef __cplusplus
}
#endif

#endif  // TI_AUTH_H_
