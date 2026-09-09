package storage

import (
	"context"
	"errors"
	"fmt"
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
	file = newRecordingFile(path, time.Duration(duration)*time.Millisecond)
	err := nativeError(code)
	if destroyed {
		t.file, t.err, t.done, t.native = file, err, true, nil
		if t.replay != nil {
			t.replay.mu.Lock()
			delete(t.replay.tasks, t)
			t.replay.mu.Unlock()
			t.replay = nil
		}
	}
	return file, err
}

type exportTaskNative interface {
	Progress() (native.CloudStorageExportProgress, int32)
	Cancel() int32
	Wait() (string, int64, int32)
	Report() (native.CloudStorageExportReport, int32)
	Close() int32
}

type ExportTask struct {
	op          nativeOperationGate
	mu          sync.Mutex
	native      exportTaskNative
	queue       *callbackQueue
	client      *Client
	device      *native.CloudStorage
	waitOnce    sync.Once
	contextOnce sync.Once
	contextDone chan struct{}
	result      ExportResult
	resultErr   error
	progress    ExportProgress
	terminal    bool
	cancelled   bool
	contextErr  error
	stopContext func() bool
}

func recordingRangeFromNative(value native.CloudStorageRange) RecordingRange {
	return RecordingRange{StartTime: time.UnixMilli(value.StartMS).UTC(), EndTime: time.UnixMilli(value.EndMS).UTC()}
}

func recordingGapFromNative(value native.CloudStorageRecordingGap) RecordingGap {
	gap := RecordingGap{Range: recordingRangeFromNative(value.Range), Tracks: make([]RecordingTrack, 0, len(value.Tracks)), Reasons: make([]RecordingGapReason, 0, len(value.Reasons))}
	for _, track := range value.Tracks {
		gap.Tracks = append(gap.Tracks, RecordingTrack{Kind: RecordingTrackKind(track.Kind), ChannelID: track.ChannelID})
	}
	for _, reason := range value.Reasons {
		gap.Reasons = append(gap.Reasons, RecordingGapReason(reason))
	}
	return gap
}

func exportReportFromNative(value native.CloudStorageExportReport) ExportReport {
	report := ExportReport{
		RequestedRange:  recordingRangeFromNative(value.RequestedRange),
		CoveredDuration: time.Duration(value.CoveredDurationMS) * time.Millisecond,
		Segments:        make([]ExportSegment, 0, len(value.Segments)), Gaps: make([]RecordingGap, 0, len(value.Gaps)),
		UnprocessedRanges: make([]RecordingRange, 0, len(value.Unprocessed)), Complete: value.Complete,
		Termination: ExportTermination(value.Termination), Cause: nativeError(value.Cause),
	}
	for _, segment := range value.Segments {
		report.Segments = append(report.Segments, ExportSegment{SourceRange: recordingRangeFromNative(segment.SourceRange), OutputStart: time.Duration(segment.OutputStartMS) * time.Millisecond, OutputEnd: time.Duration(segment.OutputEndMS) * time.Millisecond})
	}
	for _, gap := range value.Gaps {
		report.Gaps = append(report.Gaps, recordingGapFromNative(gap))
	}
	for _, item := range value.Unprocessed {
		report.UnprocessedRanges = append(report.UnprocessedRanges, recordingRangeFromNative(item))
	}
	return report
}

func (c *Client) ExportRecording(ctx context.Context, deviceID string, options ExportOptions) (*ExportTask, error) {
	if ctx == nil || deviceID == "" {
		return nil, ErrInvalidArgument
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	start, end := unixMilliseconds(options.StartTime), unixMilliseconds(options.EndTime)
	if start < 0 || start >= end {
		return nil, ErrInvalidArgument
	}
	audio := int32(-1)
	if options.AudioChannelID != nil {
		audio = int32(*options.AudioChannelID)
	}
	if c == nil {
		return nil, ErrClosed
	}
	c.op.enter()
	device, err := c.openDeviceLocked(deviceID)
	if err != nil {
		c.op.leave()
		return nil, err
	}
	task := &ExportTask{client: c, device: device, contextDone: make(chan struct{}), queue: newCallbackQueue()}
	nativeTask, code := device.Export(start, end, int32(options.VideoChannelID), audio, native.CloudStorageExportCallbacks{
		OnProgress: func(value native.CloudStorageExportProgress) {
			progress := ExportProgress{Fraction: value.Fraction, CoveredDuration: time.Duration(value.CoveredDurationMS) * time.Millisecond}
			task.mu.Lock()
			if !task.terminal && progress.Fraction >= task.progress.Fraction {
				task.progress = progress
			}
			task.mu.Unlock()
			if options.OnProgress != nil {
				_ = task.queue.post(func() { options.OnProgress(progress) })
			}
		},
		OnGap: func(value native.CloudStorageRecordingGap) {
			gap := recordingGapFromNative(value)
			if options.OnRecordingGap != nil {
				_ = task.queue.postReliable(func() { options.OnRecordingGap(gap) })
			}
		},
		OnCompleted: func(code int32, _ string, _ int64) {
			logCloudStorageResult("cloud_storage_export_terminal", nativeError(code))
		},
	})
	if code != 0 {
		task.queue.close()
		_ = c.closeDevice(device)
		c.op.leave()
		err = nativeError(code)
		logCloudStorageResult("cloud_storage_export_start", err)
		return nil, err
	}
	task.native = nativeTask
	task.stopContext = context.AfterFunc(ctx, func() {
		task.op.enter()
		defer task.op.leave()
		defer task.contextOnce.Do(func() { close(task.contextDone) })
		task.mu.Lock()
		if task.terminal {
			task.mu.Unlock()
			return
		}
		task.contextErr, task.cancelled = ctx.Err(), true
		handle := task.native
		task.mu.Unlock()
		if handle != nil {
			_ = handle.Cancel()
		}
	})
	c.registerChildLocked(task)
	c.op.leave()
	logCloudStorageResult("cloud_storage_export_start", nil)
	return task, nil
}

func (t *ExportTask) Progress() ExportProgress {
	if t == nil {
		return ExportProgress{}
	}
	t.op.enter()
	defer t.op.leave()
	t.mu.Lock()
	handle, value := t.native, t.progress
	t.mu.Unlock()
	if handle != nil {
		if current, code := handle.Progress(); code == 0 {
			value = ExportProgress{Fraction: current.Fraction, CoveredDuration: time.Duration(current.CoveredDurationMS) * time.Millisecond}
			t.mu.Lock()
			if !t.terminal {
				t.progress = value
			}
			t.mu.Unlock()
		}
	}
	return value
}

func (t *ExportTask) Cancel() error {
	if t == nil {
		return ErrClosed
	}
	t.op.enter()
	defer t.op.leave()
	t.mu.Lock()
	if t.terminal {
		t.mu.Unlock()
		return nil
	}
	t.cancelled = true
	handle := t.native
	t.mu.Unlock()
	if handle == nil {
		return ErrClosed
	}
	err := nativeError(handle.Cancel())
	logCloudStorageResult("cloud_storage_export_cancel", err)
	return err
}

func (t *ExportTask) finishWait() {
	t.mu.Lock()
	handle, client, device, stopContext := t.native, t.client, t.device, t.stopContext
	t.mu.Unlock()
	if handle == nil {
		t.mu.Lock()
		t.resultErr, t.terminal = ErrClosed, true
		t.mu.Unlock()
		return
	}
	path, duration, code := handle.Wait()
	if stopContext != nil && stopContext() {
		t.contextOnce.Do(func() { close(t.contextDone) })
	}
	<-t.contextDone
	t.op.enter()
	nativeReport, reportCode := handle.Report()
	result := ExportResult{Report: exportReportFromNative(nativeReport)}
	err := nativeError(code)
	if code == 0 {
		file := newRecordingFile(path, time.Duration(duration)*time.Millisecond)
		result.File = &file
	}
	if reportCode != 0 && err == nil {
		err = nativeError(reportCode)
	}
	closeErr := nativeError(handle.Close())
	deviceErr := client.closeDevice(device)
	if err == nil {
		if closeErr != nil {
			err = closeErr
		} else if deviceErr != nil {
			err = deviceErr
		}
	}
	t.mu.Lock()
	if err != nil && (errors.Is(err, ErrStopped) || errors.Is(err, ErrCancelled)) {
		if t.contextErr != nil {
			err = fmt.Errorf("%w: %w", ErrCancelled, withCancellationContext(err, t.contextErr))
		} else if t.cancelled {
			err = fmt.Errorf("%w: %w", ErrCancelled, err)
		}
	}
	t.result, t.resultErr = result, err
	t.progress.CoveredDuration = result.Report.CoveredDuration
	t.native, t.device, t.client, t.stopContext, t.terminal = nil, nil, nil, nil, true
	t.mu.Unlock()
	t.op.leave()
	t.queue.close()
	client.unregisterChild(t)
}

func cloneExportResult(value ExportResult) ExportResult {
	result := value
	if value.File != nil {
		file := *value.File
		result.File = &file
	}
	result.Report.Segments = append([]ExportSegment(nil), value.Report.Segments...)
	result.Report.UnprocessedRanges = append([]RecordingRange(nil), value.Report.UnprocessedRanges...)
	result.Report.Gaps = make([]RecordingGap, len(value.Report.Gaps))
	for index, gap := range value.Report.Gaps {
		result.Report.Gaps[index] = gap
		result.Report.Gaps[index].Tracks = append([]RecordingTrack(nil), gap.Tracks...)
		result.Report.Gaps[index].Reasons = append([]RecordingGapReason(nil), gap.Reasons...)
	}
	return result
}

func (t *ExportTask) Wait() (ExportResult, error) {
	if t == nil {
		return ExportResult{}, ErrClosed
	}
	if t.queue != nil && t.queue.inHandler() {
		return ExportResult{}, ErrInUse
	}
	t.waitOnce.Do(t.finishWait)
	t.mu.Lock()
	result, err := cloneExportResult(t.result), t.resultErr
	t.mu.Unlock()
	logCloudStorageResult("cloud_storage_export_wait", err)
	return result, err
}

func (t *ExportTask) preflightClientClose() error {
	if t.queue != nil && t.queue.inHandler() {
		return ErrInUse
	}
	return nil
}

func (t *ExportTask) closeFromClient() error {
	if err := t.Cancel(); err != nil && !errors.Is(err, ErrClosed) {
		return err
	}
	_, err := t.Wait()
	if errors.Is(err, ErrInUse) {
		return err
	}
	return nil
}
