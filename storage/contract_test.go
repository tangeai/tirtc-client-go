package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStableErrorMapping(t *testing.T) {
	for code, sentinel := range map[int32]error{
		6114: ErrIO,
		6115: ErrCancelled,
		6117: ErrRangeTooLarge,
		6122: ErrRecordingUnreadable,
		6123: ErrUnavailable,
		6124: ErrStopped,
		6134: ErrRecordingNotFound,
		6135: ErrRecordingDownloadFailed,
	} {
		if !errors.Is(nativeError(code), sentinel) {
			t.Fatalf("code %d did not map to %v", code, sentinel)
		}
	}
}

func TestReplayTimeConversionUsesUTCMilliseconds(t *testing.T) {
	value := time.Date(2026, 8, 13, 10, 20, 30, 987654321, time.FixedZone("fixture", 8*60*60))
	got := time.UnixMilli(unixMilliseconds(value)).UTC()
	if got.Location() != time.UTC || got.UnixMilli() != value.UnixMilli() || got.Nanosecond()%int(time.Millisecond) != 0 {
		t.Fatalf("unexpected conversion: %s", got)
	}
}

func TestOutputStateProjectionIncludesCloudStorageTerminals(t *testing.T) {
	if OutputPaused != 4 || OutputCompleted != 5 {
		t.Fatalf("unexpected CloudStorage output states: paused=%d completed=%d", OutputPaused, OutputCompleted)
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

func TestInitPreservesPathErrorForFilesystemFailure(t *testing.T) {
	root := t.TempDir()
	blockedParent := filepath.Join(root, "file")
	if err := os.WriteFile(blockedParent, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Init(InitOptions{AppID: "app", CacheDir: filepath.Join(blockedParent, "cache")})
	if !errors.Is(err, ErrIO) {
		t.Fatalf("Init does not preserve ErrIO: %v", err)
	}
	var pathError *os.PathError
	if !errors.As(err, &pathError) {
		t.Fatalf("Init does not preserve *os.PathError: %T %v", err, err)
	}
}
