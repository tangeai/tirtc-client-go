package storage

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type retryRecordingNative struct {
	mu    sync.Mutex
	calls int
}

func (n *retryRecordingNative) Stop() (string, int64, int32, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls++
	if n.calls == 1 {
		return "/cache/replay.mp4", 1200, 6026, false
	}
	return "/cache/replay.mp4", 1200, 0, true
}

func TestRecordingTaskRetriesNativeDestroy(t *testing.T) {
	nativeTask := &retryRecordingNative{}
	task := &RecordingTask{native: nativeTask}
	if _, err := task.Stop(); !errors.Is(err, ErrInUse) {
		t.Fatalf("first stop error = %v, want ErrInUse", err)
	}
	file, err := task.Stop()
	if err != nil {
		t.Fatalf("retry stop: %v", err)
	}
	if file.Path != "/cache/replay.mp4" || file.Duration != 1200*time.Millisecond {
		t.Fatalf("unexpected file: %+v", file)
	}
	if _, err := task.Stop(); err != nil {
		t.Fatalf("cached stop: %v", err)
	}
	if nativeTask.calls != 2 {
		t.Fatalf("native stop calls = %d, want 2", nativeTask.calls)
	}
}

type retryExportNative struct {
	mu         sync.Mutex
	closeCodes []int32
	closeCalls int
}

func (n *retryExportNative) Stop() (string, int64, int32) { return "", 0, 6124 }
func (n *retryExportNative) Close() int32 {
	n.mu.Lock()
	defer n.mu.Unlock()
	code := int32(0)
	if n.closeCalls < len(n.closeCodes) {
		code = n.closeCodes[n.closeCalls]
	}
	n.closeCalls++
	return code
}

func TestExportProgressIsMonotonicBoundedAndTerminal(t *testing.T) {
	task := &ExportTask{done: make(chan struct{})}
	for _, value := range []float64{0.25, 0.1, 0.75, 2} {
		task.updateProgress(value)
	}
	if got := task.Progress(); got != 1 {
		t.Fatalf("progress = %v, want 1", got)
	}
	task.complete(exportResult{err: ErrUnavailable})
	task.updateProgress(0.5)
	if got := task.Progress(); got != 1 {
		t.Fatalf("terminal progress changed to %v", got)
	}

	success := &ExportTask{done: make(chan struct{})}
	success.updateProgress(0.4)
	success.complete(exportResult{})
	if got := success.Progress(); got != 1 {
		t.Fatalf("successful terminal progress = %v, want 1", got)
	}
}

func TestRecordingTaskReleasesReplayDependencyOnlyAfterDestroy(t *testing.T) {
	nativeTask := &retryRecordingNative{}
	replay := &Replay{tasks: 1}
	task := &RecordingTask{native: nativeTask, replay: replay}
	if _, err := task.Stop(); !errors.Is(err, ErrInUse) {
		t.Fatalf("first stop error = %v, want ErrInUse", err)
	}
	if replay.tasks != 1 {
		t.Fatalf("retryable stop released replay dependency: %d", replay.tasks)
	}
	if _, err := task.Stop(); err != nil {
		t.Fatal(err)
	}
	if replay.tasks != 0 {
		t.Fatalf("destroyed task retained replay dependency: %d", replay.tasks)
	}
}

func TestExportTaskDestroyRetriesAfterInUse(t *testing.T) {
	done := make(chan struct{})
	close(done)
	nativeTask := &retryExportNative{closeCodes: []int32{6026, 0}}
	task := &ExportTask{
		native:   nativeTask,
		done:     done,
		result:   exportResult{file: RecordingFile{Path: "/cache/export.mp4", Duration: time.Second}},
		terminal: true,
	}
	if _, err := task.Wait(); !errors.Is(err, ErrInUse) {
		t.Fatalf("first wait error = %v, want ErrInUse", err)
	}
	file, err := task.Wait()
	if err != nil {
		t.Fatalf("retry wait: %v", err)
	}
	if file.Path != "/cache/export.mp4" || nativeTask.closeCalls != 2 {
		t.Fatalf("unexpected retry result: file=%+v close_calls=%d", file, nativeTask.closeCalls)
	}
}

func TestZeroValueTasksDoNotPanicOrBlock(t *testing.T) {
	if _, err := new(RecordingTask).Stop(); !errors.Is(err, ErrClosed) {
		t.Fatalf("zero RecordingTask error = %v, want ErrClosed", err)
	}
	if _, err := new(ExportTask).Wait(); !errors.Is(err, ErrClosed) {
		t.Fatalf("zero ExportTask Wait error = %v, want ErrClosed", err)
	}
	if _, err := new(ExportTask).Stop(); !errors.Is(err, ErrClosed) {
		t.Fatalf("zero ExportTask Stop error = %v, want ErrClosed", err)
	}
}
