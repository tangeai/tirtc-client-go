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

typedef struct TiRawDump TiRawDump;

typedef uint32_t TiRawDumpStopReason;
#define TI_RAW_DUMP_STOP_REASON_UNKNOWN ((TiRawDumpStopReason)0)
#define TI_RAW_DUMP_STOP_REASON_USER ((TiRawDumpStopReason)1)
#define TI_RAW_DUMP_STOP_REASON_TIME_LIMIT ((TiRawDumpStopReason)2)
#define TI_RAW_DUMP_STOP_REASON_BYTE_LIMIT ((TiRawDumpStopReason)3)
#define TI_RAW_DUMP_STOP_REASON_SOURCE_CLOSED ((TiRawDumpStopReason)4)
#define TI_RAW_DUMP_STOP_REASON_RESOURCE_LIMIT ((TiRawDumpStopReason)5)
#define TI_RAW_DUMP_STOP_REASON_WRITE_FAILED ((TiRawDumpStopReason)6)

typedef struct TiRawDumpArchive {
  // Runtime-owned strings remain valid until ti_raw_dump_destroy(). The archive
  // file is cache-managed and may be removed by a later successful capture or
  // successful ti_logging_upload(). Copy it first when longer retention is needed.
  const char* capture_id;
  const char* file_path;
  uint64_t file_size;
  const char* sha256;
  uint64_t captured_duration_ms;
  uint8_t capture_complete;
  TiRawDumpStopReason stop_reason;
  uint64_t unsaved_packet_count;
  uint64_t unsaved_byte_count;
  uint8_t empty;
} TiRawDumpArchive;

#define TI_RAW_DUMP_ARCHIVE_INITIALIZER \
  {NULL, NULL, 0, NULL, 0, 0, TI_RAW_DUMP_STOP_REASON_UNKNOWN, 0, 0, 1u}

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
// An installed sink receives DEBUG, INFO, WARNING, and ERROR and is the sole
// non-file output owner. A successful clear or replacement is a barrier for
// the previous callback generation and its user_data. Without a sink, Runtime
// writes the same four levels to its platform console only when the shared
// console option is enabled. Persistent logs and ordinary upload contain INFO
// and above; DEBUG is never persisted by this API.
TI_API TiError TI_CALL ti_logging_set_sink(const TiLogSinkOptions* options);
TI_API TiError TI_CALL ti_logging_write(TiLogLevel level, const char* tag, const char* message);
TI_API TiError TI_CALL ti_logging_upload(char* out_log_id, uint32_t out_log_id_capacity);
TI_API TiError TI_CALL ti_raw_dump_stop(TiRawDump* dump, TiRawDumpArchive* out_archive);
TI_API TiError TI_CALL ti_raw_dump_destroy(TiRawDump* dump);

#ifdef __cplusplus
}
#endif

#endif  // TI_RUNTIME_H_
