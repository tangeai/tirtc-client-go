#ifndef TI_RUNTIME_H_
#define TI_RUNTIME_H_

#include <stddef.h>
#include <stdint.h>

#include "ti/error.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct TiBuildInfo {
  const char* semver_base;
  const char* semver_effective;
  const char* commit_hash;
  const char* build_branch;
  const char* build_time_utc;
  const char* version_string;
  uint32_t commit_count;
} TiBuildInfo;

typedef void(TI_CALL* TiCallbackTaskFn)(void* task_data);
typedef void(TI_CALL* TiCallbackDispatchFn)(TiCallbackTaskFn task, void* task_data,
                                            void* user_data);

// A NULL dispatch selects synchronous borrowed callbacks and requires NULL
// user_data. Otherwise dispatch may execute the Runtime-owned opaque task
// immediately or later, but it must execute every task exactly once. The task
// and task_data pair must not be changed, freed, reused after task returns, or
// retained forever. Callback clear/replacement and object destruction return
// TI_ERROR_IN_USE until all accepted tasks have returned.
typedef struct TiCallbackDispatcher {
  TiCallbackDispatchFn dispatch;
  void* user_data;
} TiCallbackDispatcher;

#define TI_CALLBACK_DISPATCHER_INITIALIZER {NULL, NULL}

typedef uint32_t TiLogLevel;
#define TI_LOG_ID_CAPACITY 256u
#define TI_LOG_LEVEL_DEBUG ((TiLogLevel)0)
#define TI_LOG_LEVEL_INFO ((TiLogLevel)1)
#define TI_LOG_LEVEL_WARNING ((TiLogLevel)2)
#define TI_LOG_LEVEL_ERROR ((TiLogLevel)3)

typedef void(TI_CALL* TiLogSinkFn)(TiLogLevel level, const char* tag, const char* message,
                                   void* user_data);

typedef struct TiLogSinkOptions {
  TiLogSinkFn on_log;
  void* user_data;
  TiCallbackDispatcher dispatcher;
} TiLogSinkOptions;

#define TI_LOG_SINK_OPTIONS_INITIALIZER {NULL, NULL, TI_CALLBACK_DISPATCHER_INITIALIZER}

// Returned strings are Runtime-owned, NUL-terminated, and remain valid until
// the process unloads the Runtime aggregate.
TI_API const TiBuildInfo* TI_CALL ti_build_info_get(void);
TI_API const char* TI_CALL ti_build_version_string(void);
// A successful clear or replacement is a barrier for the previous callback
// generation and its user_data.
TI_API TiError TI_CALL ti_logging_set_sink(const TiLogSinkOptions* options);
TI_API TiError TI_CALL ti_logging_write(TiLogLevel level, const char* tag, const char* message);
TI_API TiError TI_CALL ti_logging_upload(char* out_log_id, uint32_t out_log_id_capacity);

#ifdef __cplusplus
}
#endif

#endif  // TI_RUNTIME_H_
