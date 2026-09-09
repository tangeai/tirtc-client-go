package storage

import (
	"errors"
	"testing"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/buildidentity"
)

func TestPublicClientConsumesCloudStorageBuildIdentity(t *testing.T) {
	buildidentity.Release("cloud_storage")
	client, err := NewClient(ClientOptions{AppID: "app", AccessKeyID: "key", AccessKeySecret: "secret", CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, available := buildidentity.Line("cloud_storage"); available {
		t.Fatal("Cloud Storage public Init did not consume the build identity")
	}
}

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

func TestClientRejectsRelativeCachePath(t *testing.T) {
	_, err := NewClient(ClientOptions{AppID: "app", AccessKeyID: "key", AccessKeySecret: "secret", CacheDir: "relative"})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("Client = %v", err)
	}
}
