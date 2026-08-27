package storage_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/storage"
)

func TestPublicAPIDump(t *testing.T) {
	want, err := os.ReadFile("testdata/public_api.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command("go", "doc", "-all", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go doc: %v\n%s", err, got)
	}
	got = append(bytes.TrimRight(got, "\n"), '\n')
	if !bytes.Equal(got, want) {
		t.Fatal("Ti Cloud Storage public API differs from testdata/public_api.txt; review the contract and regenerate the dump")
	}
	for _, name := range []string{
		"ConnService", "Input", "Metrics", "LinkMode", "Log", "LogLevel", "OnSizeChanged", "TokenExpiredHandler",
	} {
		if regexp.MustCompile(`\b` + name + `\b`).Match(got) {
			t.Fatalf("forbidden public identifier %s is present", name)
		}
	}
}

func TestPublicAPISignatures(t *testing.T) {
	var _ func(storage.InitOptions) error = storage.Init
	var _ func() error = storage.Shutdown
	var _ func(string) (*storage.CloudStorage, error) = storage.New
	var _ func(*storage.CloudStorage, context.Context, time.Time, time.Time) ([]storage.RecordingRange, error) = (*storage.CloudStorage).ListRecordings
	var _ func(*storage.CloudStorage, context.Context, string, string) ([]storage.RecordingDay, error) = (*storage.CloudStorage).ListRecordingDays
	var _ func(*storage.CloudStorage, context.Context, string, string, string) ([]storage.RecordingDay, error) = (*storage.CloudStorage).ListRecordingDaysInTimeZone
	var _ func(*storage.CloudStorage, string) error = (*storage.CloudStorage).UpdateToken
	var _ func(*storage.CloudStorage, storage.ReplayOptions) (*storage.Replay, error) = (*storage.CloudStorage).NewReplay
	var _ func(*storage.CloudStorage, storage.ExportOptions) (*storage.ExportTask, error) = (*storage.CloudStorage).ExportRecording
	var _ func(*storage.CloudStorage) error = (*storage.CloudStorage).Close

	var _ func(*storage.Replay, time.Time, time.Time) error = (*storage.Replay).Play
	var _ func(*storage.Replay, time.Time, time.Time, time.Time) error = (*storage.Replay).PlayAt
	var _ func(*storage.Replay) error = (*storage.Replay).Pause
	var _ func(*storage.Replay) error = (*storage.Replay).Resume
	var _ func(*storage.Replay, time.Time) error = (*storage.Replay).Seek
	var _ func(*storage.Replay, storage.ReplaySpeed) error = (*storage.Replay).SetSpeed
	var _ func(*storage.Replay) storage.ReplaySpeed = (*storage.Replay).Speed
	var _ func(*storage.Replay) (time.Time, bool, error) = (*storage.Replay).CurrentTime
	var _ func(*storage.Replay) error = (*storage.Replay).Stop
	var _ func(*storage.Replay, storage.StartRecordingOptions) (*storage.RecordingTask, error) = (*storage.Replay).StartRecording
	var _ func(*storage.Replay) error = (*storage.Replay).Close

	_ = storage.InitOptions{AppID: "", CacheDir: "", Endpoint: "", ConsoleLogEnabled: false}
	_ = storage.ReplayOptions{OnTimeChanged: func(time.Time) {}, OnCompleted: func() {}, OnError: func(error) {}}
	_ = storage.ExportOptions{StartTime: time.Time{}, EndTime: time.Time{}, VideoChannelID: 0, AudioChannelID: nil}
	_ = storage.StartRecordingOptions{VideoChannelID: 0, AudioChannelID: nil}
}
