package storage

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
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
	replay := &Replay{tasks: make(map[*RecordingTask]struct{})}
	task := &RecordingTask{native: nativeTask, replay: replay}
	replay.tasks[task] = struct{}{}
	if _, err := task.Stop(); !errors.Is(err, ErrInUse) {
		t.Fatalf("first stop error = %v", err)
	}
	if len(replay.tasks) != 1 {
		t.Fatal("retryable stop released Replay")
	}
	file, err := task.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if file.Path != "/cache/replay.mp4" || file.Duration != 1200*time.Millisecond {
		t.Fatalf("file = %+v", file)
	}
	if len(replay.tasks) != 0 || nativeTask.calls != 2 {
		t.Fatalf("tasks=%d calls=%d", len(replay.tasks), nativeTask.calls)
	}
}

func TestExportResultCloneIsolatedFromCallerMutation(t *testing.T) {
	original := ExportResult{File: &RecordingFile{Path: "/cache/export.mp4"}, Report: ExportReport{
		Segments:          []ExportSegment{{OutputEnd: time.Second}},
		Gaps:              []RecordingGap{{Tracks: []RecordingTrack{{Kind: RecordingTrackVideo, ChannelID: 1}}, Reasons: []RecordingGapReason{RecordingGapNotFound}}},
		UnprocessedRanges: []RecordingRange{{StartTime: time.Unix(1, 0)}},
	}}
	copy := cloneExportResult(original)
	copy.File.Path = "changed"
	copy.Report.Segments[0].OutputEnd = 0
	copy.Report.Gaps[0].Tracks[0].ChannelID = 9
	copy.Report.Gaps[0].Reasons[0] = RecordingGapUnknown
	if original.File.Path != "/cache/export.mp4" || original.Report.Segments[0].OutputEnd != time.Second || original.Report.Gaps[0].Tracks[0].ChannelID != 1 || original.Report.Gaps[0].Reasons[0] != RecordingGapNotFound {
		t.Fatal("clone shares mutable result storage")
	}
}

func TestZeroValueExportTaskDoesNotBlock(t *testing.T) {
	if _, err := new(ExportTask).Wait(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Wait = %v", err)
	}
	if err := new(ExportTask).Cancel(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Cancel = %v", err)
	}
}

// The private Native boundary holds Close so calls made during finalization can be observed.
type exportFinalizationNative struct {
	closeEntered    chan struct{}
	releaseClose    chan struct{}
	closed          atomic.Bool
	callsAfterClose atomic.Int32
}

func (n *exportFinalizationNative) Progress() (native.CloudStorageExportProgress, int32) {
	if n.closed.Load() {
		n.callsAfterClose.Add(1)
	}
	return native.CloudStorageExportProgress{Fraction: 0.4, CoveredDurationMS: 400}, 0
}
func (n *exportFinalizationNative) Cancel() int32 {
	if n.closed.Load() {
		n.callsAfterClose.Add(1)
	}
	return 0
}
func (n *exportFinalizationNative) Wait() (string, int64, int32) {
	return "/cache/partial.mp4", 400, 0
}
func (n *exportFinalizationNative) Report() (native.CloudStorageExportReport, int32) {
	return native.CloudStorageExportReport{CoveredDurationMS: 400, Complete: false}, 0
}
func (n *exportFinalizationNative) Close() int32 {
	n.closed.Store(true)
	close(n.closeEntered)
	<-n.releaseClose
	return 0
}

func TestExportFinalizationKeepsNativeProgressAndSerializesHandleCalls(t *testing.T) {
	handle := &exportFinalizationNative{closeEntered: make(chan struct{}), releaseClose: make(chan struct{})}
	client := &Client{children: make(map[clientChild]struct{})}
	task := &ExportTask{native: handle, client: client, queue: newCallbackQueue(), contextDone: make(chan struct{}),
		progress: ExportProgress{Fraction: 0.4, CoveredDuration: 400 * time.Millisecond}}
	close(task.contextDone)
	client.children[task] = struct{}{}
	waited := make(chan error, 1)
	go func() { _, err := task.Wait(); waited <- err }()
	<-handle.closeEntered
	progressDone := make(chan ExportProgress, 1)
	cancelDone := make(chan error, 1)
	go func() { progressDone <- task.Progress() }()
	go func() { cancelDone <- task.Cancel() }()
	// Both calls must wait for handle retirement, then use terminal Go state.
	progressReturned, cancelReturned := false, false
	select {
	case <-progressDone:
		progressReturned = true
		t.Error("Progress entered a closing native handle")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case <-cancelDone:
		cancelReturned = true
		t.Error("Cancel entered a closing native handle")
	case <-time.After(20 * time.Millisecond):
	}
	close(handle.releaseClose)
	if err := <-waited; err != nil {
		t.Fatal(err)
	}
	if !progressReturned {
		<-progressDone
	}
	if !cancelReturned {
		if err := <-cancelDone; err != nil {
			t.Fatal(err)
		}
	}
	if got := task.Progress(); got.Fraction != 0.4 || got.CoveredDuration != 400*time.Millisecond {
		t.Fatalf("partial progress = %+v", got)
	}
	if handle.callsAfterClose.Load() != 0 {
		t.Fatal("native use overlapped Close")
	}
}

func TestExportWaitDrainsOtherGoroutineHandlerAndRejectsSelfWait(t *testing.T) {
	handle := &exportFinalizationNative{closeEntered: make(chan struct{}), releaseClose: make(chan struct{})}
	close(handle.releaseClose)
	client := &Client{children: make(map[clientChild]struct{})}
	task := &ExportTask{native: handle, client: client, queue: newCallbackQueue(), contextDone: make(chan struct{})}
	close(task.contextDone)
	client.children[task] = struct{}{}
	entered, release := make(chan error, 1), make(chan struct{})
	task.queue.post(func() {
		_, err := task.Wait()
		entered <- err
		<-release
	})
	selfErr := <-entered
	if !errors.Is(selfErr, ErrInUse) {
		t.Errorf("self Wait = %v", selfErr)
	}
	if task.queue.inHandler() {
		t.Error("another goroutine was identified as the handler")
	}
	waited := make(chan error, 1)
	go func() { _, err := task.Wait(); waited <- err }()
	early := false
	select {
	case err := <-waited:
		early = true
		t.Errorf("external Wait returned before drain: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if !early {
		if err := <-waited; err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecordingFileCopiesShareSuccessfulDeletion(t *testing.T) {
	cache := t.TempDir()
	client, err := NewClient(ClientOptions{AppID: "file-owner", AccessKeyID: "key", AccessKeySecret: "secret", CacheDir: cache})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	directory := filepath.Join(cache, "media", "cloud_storage-recordings")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "export-1-1.mp4")
	if err := os.WriteFile(path, []byte("owned-file"), 0600); err != nil {
		t.Fatal(err)
	}
	result := ExportResult{File: func() *RecordingFile { f := newRecordingFile(path, time.Second); return &f }()}
	copied := cloneExportResult(result)
	if err := result.File.Delete(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := copied.File.Delete(); err != nil {
		t.Fatalf("copied terminal file lost successful deletion: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file still exists: %v", err)
	}
}
