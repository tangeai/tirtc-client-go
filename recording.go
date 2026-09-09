package tirtc

import (
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type RecordingTask struct {
	mu         sync.Mutex
	native     *native.RecordingTask
	connection *Conn
	stopping   bool
	done       chan struct{}
	finished   bool
	file       RecordingFile
	err        error
}

func (c *Conn) StartRecording(options StartRecordingOptions) (*RecordingTask, error) {
	if !validStreamID(options.VideoStreamID) ||
		(options.AudioStreamID != nil && (!validStreamID(*options.AudioStreamID) || *options.AudioStreamID == options.VideoStreamID)) {
		return nil, ErrInvalidArgument
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.mu.Lock()
	if c.closed || c.native == nil {
		c.mu.Unlock()
		return nil, ErrClosed
	}
	handle := c.native
	c.mu.Unlock()
	audio := int32(-1)
	if options.AudioStreamID != nil {
		audio = int32(*options.AudioStreamID)
	}
	nativeTask, code := handle.StartRecording(int32(options.VideoStreamID), audio)
	if code != 0 {
		err := nativeError(code)
		logSDKResult("recording_start", err)
		return nil, err
	}
	logSDKResult("recording_start", nil)
	task := &RecordingTask{native: nativeTask, connection: c}
	c.mu.Lock()
	c.tasks[task] = struct{}{}
	c.mu.Unlock()
	return task, nil
}

func (t *RecordingTask) Stop() (file RecordingFile, resultErr error) {
	defer func() { logSDKResult("recording_stop", resultErr) }()
	if t == nil {
		return RecordingFile{}, ErrClosed
	}
	for {
		t.mu.Lock()
		if t.finished {
			file, err := t.file, t.err
			t.mu.Unlock()
			return file, err
		}
		if t.stopping {
			done := t.done
			t.mu.Unlock()
			<-done
			continue
		}
		t.stopping = true
		t.done = make(chan struct{})
		done := t.done
		handle := t.native
		t.mu.Unlock()

		path, durationMs, code, destroyed := handle.Stop()
		file := RecordingFile{Path: path, Duration: time.Duration(durationMs) * time.Millisecond}
		err := nativeError(code)

		t.mu.Lock()
		if destroyed {
			t.finished = true
			t.native = nil
			t.file = file
			t.err = err
			t.connection.mu.Lock()
			delete(t.connection.tasks, t)
			t.connection.mu.Unlock()
		}
		t.stopping = false
		close(done)
		t.mu.Unlock()
		return file, err
	}
}
