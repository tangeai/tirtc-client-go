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
	var _ func(storage.ClientOptions) (*storage.Client, error) = storage.NewClient
	var _ func(*storage.Client, context.Context, string, time.Time, time.Time) ([]storage.RecordingRange, error) = (*storage.Client).ListRecordings
	var _ func(*storage.Client, context.Context, string, string, string) ([]storage.RecordingDay, error) = (*storage.Client).ListRecordingDays
	var _ func(*storage.Client, context.Context, string, string, string, string) ([]storage.RecordingDay, error) = (*storage.Client).ListRecordingDaysInTimeZone
	var _ func(*storage.Client, string, storage.ReplayOptions) (*storage.Replay, error) = (*storage.Client).NewReplay
	var _ func(*storage.Client, context.Context, string, storage.ExportOptions) (*storage.ExportTask, error) = (*storage.Client).ExportRecording
	var _ func(*storage.Client) error = (*storage.Client).Close

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

	var _ func(*storage.ExportTask) storage.ExportProgress = (*storage.ExportTask).Progress
	var _ func(*storage.ExportTask) (storage.ExportResult, error) = (*storage.ExportTask).Wait
	var _ func(*storage.ExportTask) error = (*storage.ExportTask).Cancel
	_ = storage.ClientOptions{AppID: "", AccessKeyID: "", AccessKeySecret: "", CacheDir: "", Endpoint: "", ConsoleLogEnabled: false}
	_ = storage.ReplayOptions{OnTimeChanged: func(time.Time) {}, OnCompleted: func() {}, OnError: func(error) {}, OnRecordingGap: func(storage.RecordingGap) {}}
	_ = storage.ExportOptions{StartTime: time.Time{}, EndTime: time.Time{}, VideoChannelID: 0, AudioChannelID: nil, OnProgress: func(storage.ExportProgress) {}, OnRecordingGap: func(storage.RecordingGap) {}}
	_ = storage.StartRecordingOptions{VideoChannelID: 0, AudioChannelID: nil}
}
