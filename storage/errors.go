package storage

import (
	"errors"
	"fmt"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

var (
	ErrInvalidArgument         = errors.New("storage: invalid argument")
	ErrNotInitialized          = errors.New("storage: not initialized")
	ErrTokenExpired            = errors.New("storage: token expired")
	ErrAlreadyInitialized      = errors.New("storage: already initialized")
	ErrPermissionDenied        = errors.New("storage: permission denied")
	ErrInUse                   = errors.New("storage: resource in use")
	ErrNotStarted              = errors.New("storage: not started")
	ErrNotBound                = errors.New("storage: not bound")
	ErrNotConfigured           = errors.New("storage: not configured")
	ErrResourceExhausted       = errors.New("storage: resource exhausted")
	ErrUnsupportedFormat       = errors.New("storage: unsupported format")
	ErrIO                      = errors.New("storage: I/O failed")
	ErrCancelled               = errors.New("storage: cancelled")
	ErrRangeTooLarge           = errors.New("storage: range too large")
	ErrNoFrame                 = errors.New("storage: no frame")
	ErrNoRecordableMedia       = errors.New("storage: no recordable media")
	ErrRecordingOverrun        = errors.New("storage: recording overrun")
	ErrRecordingUnreadable     = errors.New("storage: recording unreadable")
	ErrRecordingNotFound       = errors.New("storage: recording not found")
	ErrRecordingDownloadFailed = errors.New("storage: recording download failed")
	ErrUnavailable             = errors.New("storage: unavailable")
	ErrStopped                 = errors.New("storage: stopped")
	ErrClosed                  = errors.New("storage: closed")
)

type Error struct{ Code int32 }

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("storage: %s (%d)", ErrorName(e.Code), e.Code)
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
		6014: ErrTokenExpired,
		6022: ErrAlreadyInitialized,
		6024: ErrPermissionDenied,
		6026: ErrInUse,
		6027: ErrNotStarted,
		6029: ErrNotBound,
		6030: ErrNotConfigured,
		6043: ErrResourceExhausted,
		6046: ErrIO,
		6113: ErrUnsupportedFormat,
		6114: ErrIO,
		6115: ErrCancelled,
		6117: ErrRangeTooLarge,
		6118: ErrNoFrame,
		6119: ErrNoRecordableMedia,
		6120: ErrRecordingOverrun,
		6122: ErrRecordingUnreadable,
		6123: ErrUnavailable,
		6124: ErrStopped,
		6134: ErrRecordingNotFound,
		6135: ErrRecordingDownloadFailed,
	}[code]
}
