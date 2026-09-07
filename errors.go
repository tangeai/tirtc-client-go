package tirtc

import (
	"errors"
	"fmt"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

var (
	ErrInvalidArgument      = errors.New("tirtc: invalid argument")
	ErrNotInitialized       = errors.New("tirtc: not initialized")
	ErrAuthenticationFailed = errors.New("tirtc: authentication failed")
	ErrTimeout              = errors.New("tirtc: timeout")
	ErrRemoteClosed         = errors.New("tirtc: remote closed")
	ErrTokenExpired         = errors.New("tirtc: token expired")
	ErrAlreadyInitialized   = errors.New("tirtc: already initialized")
	ErrPermissionDenied     = errors.New("tirtc: permission denied")
	ErrInUse                = errors.New("tirtc: resource in use")
	ErrNotStarted           = errors.New("tirtc: not started")
	ErrNotConnected         = errors.New("tirtc: not connected")
	ErrNotBound             = errors.New("tirtc: not bound")
	ErrNotConfigured        = errors.New("tirtc: not configured")
	ErrResourceExhausted    = errors.New("tirtc: resource exhausted")
	ErrLogExport            = errors.New("tirtc: log export failed")
	ErrLogUpload            = errors.New("tirtc: log upload failed")
	ErrUnsupported          = errors.New("tirtc: unsupported")
	ErrUnsupportedFormat    = errors.New("tirtc: unsupported format")
	ErrIO                   = errors.New("tirtc: I/O failed")
	ErrNoFrame              = errors.New("tirtc: no frame")
	ErrNoRecordableMedia    = errors.New("tirtc: no recordable media")
	ErrRecordingOverrun     = errors.New("tirtc: recording overrun")
	ErrClosed               = errors.New("tirtc: closed")
)

type Error struct{ Code int32 }

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("tirtc: %s (%d)", ErrorName(e.Code), e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return errorSentinel(e.Code)
}

func ErrorName(code int32) string { return native.ErrorName(code) }

func nativeError(code int32) error {
	if code == 0 {
		return nil
	}
	return &Error{Code: code}
}

func errorSentinel(code int32) error {
	return map[int32]error{
		6000: ErrInvalidArgument,
		6001: ErrNotInitialized,
		6008: ErrAuthenticationFailed,
		6009: ErrTimeout,
		6012: ErrRemoteClosed,
		6014: ErrTokenExpired,
		6022: ErrAlreadyInitialized,
		6024: ErrPermissionDenied,
		6026: ErrInUse,
		6027: ErrNotStarted,
		6028: ErrNotConnected,
		6029: ErrNotBound,
		6030: ErrNotConfigured,
		6032: ErrInvalidArgument,
		6043: ErrResourceExhausted,
		6044: ErrIO,
		6045: ErrIO,
		6046: ErrIO,
		6048: ErrLogExport,
		6049: ErrLogUpload,
		6073: ErrLogExport,
		6107: ErrUnsupported,
		6113: ErrUnsupportedFormat,
		6114: ErrIO,
		6118: ErrNoFrame,
		6119: ErrNoRecordableMedia,
		6120: ErrRecordingOverrun,
		6121: ErrIO,
		6125: ErrLogExport,
		6126: ErrLogExport,
		6127: ErrLogUpload,
		6128: ErrLogUpload,
		6129: ErrLogUpload,
		6131: ErrLogUpload,
		6132: ErrLogUpload,
		6133: ErrLogUpload,
	}[code]
}
