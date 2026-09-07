package tirtc

import (
	"errors"
	"testing"
)

func TestNativeErrorPreservesCodeAndStableSentinel(t *testing.T) {
	err := nativeError(6032)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("errors.Is = false: %v", err)
	}
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != 6032 {
		t.Fatalf("errors.As = %#v", typed)
	}
	unknown := nativeError(6999)
	if !errors.As(unknown, &typed) || typed.Code != 6999 {
		t.Fatalf("unknown error lost its code: %v", unknown)
	}
	if errors.Unwrap(unknown) != nil {
		t.Fatalf("unknown code invented a category: %v", unknown)
	}
}

func TestLogUploadStageErrorsPreserveCodeAndStableSentinel(t *testing.T) {
	for _, code := range []int32{6131, 6132, 6133} {
		err := nativeError(code)
		if !errors.Is(err, ErrLogUpload) {
			t.Fatalf("errors.Is(%d, ErrLogUpload) = false: %v", code, err)
		}
		var typed *Error
		if !errors.As(err, &typed) || typed.Code != code {
			t.Fatalf("errors.As(%d) = %#v", code, typed)
		}
	}
}

func TestMediaFileDeleteRejectsZeroValueBeforeNativeCall(t *testing.T) {
	for name, remove := range map[string]func() error{
		"recording": RecordingFile{}.Delete,
		"snapshot":  SnapshotFile{}.Delete,
	} {
		if err := remove(); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("%s Delete error = %v", name, err)
		}
	}
}
