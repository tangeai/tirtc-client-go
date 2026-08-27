package native

import (
	"testing"
	"time"
)

func TestDestroyAfterTerminalUsesBoundedBackoff(t *testing.T) {
	now := time.Unix(0, 0)
	calls := 0
	code := destroyAfterTerminalWithClock(func() int32 {
		calls++
		return errorInUse
	}, func() time.Time { return now }, func(delay time.Duration) { now = now.Add(delay) })
	if code != errorInUse || calls < 2 || calls > 120 {
		t.Fatalf("bounded destroy result code=%d calls=%d", code, calls)
	}
	if elapsed := now.Sub(time.Unix(0, 0)); elapsed > 2*time.Second {
		t.Fatalf("bounded destroy exceeded deadline: %v", elapsed)
	}
}

func TestDestroyAfterTerminalRecoversFromTransientInUse(t *testing.T) {
	now := time.Unix(0, 0)
	calls := 0
	code := destroyAfterTerminalWithClock(func() int32 {
		calls++
		if calls < 3 {
			return errorInUse
		}
		return 0
	}, func() time.Time { return now }, func(delay time.Duration) { now = now.Add(delay) })
	if code != 0 || calls != 3 {
		t.Fatalf("transient destroy result code=%d calls=%d", code, calls)
	}
}

func TestStopAndDestroyCloudStorageRecordingTaskCopiesPathBeforeDestroy(t *testing.T) {
	const wantPath = "/tmp/tirtc-cloud_storage/recording with spaces and utf8-录制.mp4"
	const wantDuration int64 = 9876543210
	destroyed := false

	path, duration, code, released := stopAndDestroyCloudStorageRecordingTask(func() (string, int64, int32) {
		if destroyed {
			t.Fatal("recording path was read after the native task was destroyed")
		}
		return wantPath, wantDuration, 0
	}, func() int32 {
		destroyed = true
		return 0
	})

	if path != wantPath || duration != wantDuration || code != 0 || !released || !destroyed {
		t.Fatalf("unexpected stop result: path=%q duration=%d code=%d released=%v destroyed=%v",
			path, duration, code, released, destroyed)
	}
}

func TestStopAndDestroyCloudStorageRecordingTaskKeepsStopError(t *testing.T) {
	const stopError int32 = 6123
	destroyed := false

	path, duration, code, released := stopAndDestroyCloudStorageRecordingTask(func() (string, int64, int32) {
		return "ignored", 42, stopError
	}, func() int32 {
		destroyed = true
		return 0
	})

	if path != "" || duration != 0 || code != stopError || !released || !destroyed {
		t.Fatalf("unexpected error result: path=%q duration=%d code=%d released=%v destroyed=%v",
			path, duration, code, released, destroyed)
	}
}
