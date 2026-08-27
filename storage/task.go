package storage

import (
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type RecordingTask struct {
	mu     sync.Mutex
	native recordingTaskNative
	replay *Replay
	file   RecordingFile
	err    error
	done   bool
}

type recordingTaskNative interface {
	Stop() (string, int64, int32, bool)
}

func (t *RecordingTask) Stop() (file RecordingFile, resultErr error) {
	defer func() { logCloudStorageResult("cloud_storage_recording_stop", resultErr) }()
	if t == nil {
		return RecordingFile{}, ErrClosed
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return t.file, t.err
	}
	if t.native == nil {
		return RecordingFile{}, ErrClosed
	}
	path, duration, code, destroyed := t.native.Stop()
	file = RecordingFile{Path: path, Duration: time.Duration(duration) * time.Millisecond}
	err := nativeError(code)
	if destroyed {
		t.file = file
		t.err = err
		t.done = true
		t.native = nil
		if t.replay != nil {
			t.replay.mu.Lock()
			if t.replay.tasks > 0 {
				t.replay.tasks--
			}
			t.replay.mu.Unlock()
			t.replay = nil
		}
	}
	return file, err
}

type exportResult struct {
	file RecordingFile
	err  error
}

type ExportTask struct {
	op       nativeOperationGate
	mu       sync.Mutex
	native   exportTaskNative
	done     chan struct{}
	result   exportResult
	progress float64
	terminal bool
}

type exportTaskNative interface {
	Stop() (string, int64, int32)
	Close() int32
}

func (s *CloudStorage) ExportRecording(options ExportOptions) (*ExportTask, error) {
	start, end := unixMilliseconds(options.StartTime), unixMilliseconds(options.EndTime)
	if start < 0 || start >= end {
		return nil, ErrInvalidArgument
	}
	audio := int32(-1)
	if options.AudioChannelID != nil {
		audio = int32(*options.AudioChannelID)
	}
	s.op.enter()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.op.leave()
		return nil, ErrClosed
	}
	handle := s.native
	s.mu.Unlock()
	task := &ExportTask{done: make(chan struct{})}
	nativeTask, code := handle.Export(start, end, int32(options.VideoChannelID), audio,
		native.CloudStorageExportCallbacks{
			OnProgress: func(value float64) {
				task.updateProgress(value)
			},
			OnCompleted: func(code int32, path string, duration int64) {
				logCloudStorageResult("cloud_storage_export_terminal", nativeError(code))
				task.complete(exportResult{
					file: RecordingFile{Path: path, Duration: time.Duration(duration) * time.Millisecond},
					err:  nativeError(code),
				})
			},
		})
	s.op.leave()
	if code != 0 {
		err := nativeError(code)
		logCloudStorageResult("cloud_storage_export_start", err)
		return nil, err
	}
	task.native = nativeTask
	logCloudStorageResult("cloud_storage_export_start", nil)
	return task, nil
}

func (t *ExportTask) updateProgress(value float64) {
	t.mu.Lock()
	if !t.terminal && value > t.progress {
		if value > 1 {
			value = 1
		}
		t.progress = value
	}
	t.mu.Unlock()
}

func (t *ExportTask) complete(result exportResult) {
	t.mu.Lock()
	if !t.terminal {
		t.result = result
		if result.err == nil {
			t.progress = 1
		}
		t.terminal = true
		close(t.done)
	}
	t.mu.Unlock()
}

func (t *ExportTask) Progress() float64 {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	value := t.progress
	t.mu.Unlock()
	return value
}

func (t *ExportTask) destroy() error {
	if t == nil {
		return ErrClosed
	}
	t.op.enter()
	defer t.op.leave()
	t.mu.Lock()
	handle := t.native
	t.mu.Unlock()
	if handle == nil {
		return nil
	}
	if err := nativeError(handle.Close()); err != nil {
		return err
	}
	t.mu.Lock()
	if t.native == handle {
		t.native = nil
	}
	t.mu.Unlock()
	return nil
}

func (t *ExportTask) Wait() (RecordingFile, error) {
	if t == nil || t.done == nil {
		return RecordingFile{}, ErrClosed
	}
	<-t.done
	t.mu.Lock()
	result := t.result
	t.mu.Unlock()
	if err := t.destroy(); err != nil && result.err == nil {
		result.err = err
	}
	logCloudStorageResult("cloud_storage_export_wait", result.err)
	return result.file, result.err
}

func (t *ExportTask) Stop() (RecordingFile, error) {
	if t == nil || t.done == nil {
		return RecordingFile{}, ErrClosed
	}
	t.op.enter()
	t.mu.Lock()
	if t.terminal {
		t.mu.Unlock()
		t.op.leave()
		return t.Wait()
	}
	handle := t.native
	t.mu.Unlock()
	if handle == nil {
		t.op.leave()
		return RecordingFile{}, ErrClosed
	}
	path, duration, code := handle.Stop()
	t.complete(exportResult{
		file: RecordingFile{Path: path, Duration: time.Duration(duration) * time.Millisecond},
		err:  nativeError(code),
	})
	t.op.leave()
	file, err := t.Wait()
	logCloudStorageResult("cloud_storage_export_stop", err)
	return file, err
}
