package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
	"github.com/tangeai/tirtc-client-go/v2/internal/runtimelease"
)

type InitOptions struct {
	AppID             string
	CacheDir          string
	Endpoint          string
	ConsoleLogEnabled bool
}

func Init(options InitOptions) error {
	if options.AppID == "" || options.CacheDir == "" || !filepath.IsAbs(options.CacheDir) {
		return ErrInvalidArgument
	}
	options.CacheDir = filepath.Clean(options.CacheDir)
	err := runtimelease.Init(runtimelease.CloudStorage, runtimelease.Configuration{
		AppID: options.AppID, Endpoint: options.Endpoint, CacheDir: options.CacheDir,
		ConsoleLogEnabled: options.ConsoleLogEnabled,
	}, func() error {
		if err := ensureWritableDir(options.CacheDir); err != nil {
			return fmt.Errorf("%w: %w", ErrIO, err)
		}
		return nativeError(native.CloudStorageInit(native.InitOptions{
			AppID: options.AppID, Endpoint: options.Endpoint, CacheDir: options.CacheDir,
			ConsoleLogEnabled: options.ConsoleLogEnabled,
		}))
	})
	if errors.Is(err, runtimelease.ErrConflict) {
		err = ErrAlreadyInitialized
	}
	logCloudStorageResult("cloud_storage_init", err)
	return err
}

func ensureWritableDir(directory string) error {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	probe, err := os.CreateTemp(directory, ".tirtc-write-probe-")
	if err != nil {
		return err
	}
	name := probe.Name()
	if closeErr := probe.Close(); closeErr != nil {
		_ = os.Remove(name)
		return closeErr
	}
	return os.Remove(name)
}

func Shutdown() error {
	logCloudStorageEvent("cloud_storage_shutdown_started")
	err := runtimelease.Shutdown(runtimelease.CloudStorage, func() error {
		err := nativeError(native.CloudStorageShutdown())
		if errors.Is(err, ErrNotInitialized) {
			return nil
		}
		return err
	})
	if err != nil {
		logCloudStorageResult("cloud_storage_shutdown", err)
	}
	return err
}

type CloudStorage struct {
	op           nativeOperationGate
	mu           sync.Mutex
	native       *native.CloudStorage
	queue        *callbackQueue
	retainedList cloudStorageListReleaser
	listActive   bool
	closed       bool
}

type cloudStorageListReleaser interface {
	Close() int32
}

func New(token string) (*CloudStorage, error) {
	if token == "" {
		return nil, ErrInvalidArgument
	}
	cloudStorage := &CloudStorage{queue: newCallbackQueue()}
	handle, code := native.NewCloudStorage(token)
	if code != 0 {
		cloudStorage.queue.close()
		err := nativeError(code)
		logCloudStorageResult("cloud_storage_create", err)
		return nil, err
	}
	cloudStorage.native = handle
	logCloudStorageResult("cloud_storage_create", nil)
	return cloudStorage, nil
}

func (s *CloudStorage) UpdateToken(token string) error {
	if token == "" {
		return ErrInvalidArgument
	}
	s.op.enter()
	defer s.op.leave()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	handle := s.native
	s.mu.Unlock()
	err := nativeError(handle.UpdateToken(token))
	logCloudStorageResult("cloud_storage_update_token", err)
	return err
}

func (s *CloudStorage) ListRecordings(ctx context.Context, startTime, endTime time.Time) (result []RecordingRange, resultErr error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	start := unixMilliseconds(startTime)
	end := unixMilliseconds(endTime)
	if start < 0 || start >= end {
		return nil, ErrInvalidArgument
	}
	logCloudStorageEvent("cloud_storage_list_recordings_started")
	defer func() { logCloudStorageResult("cloud_storage_list_recordings", resultErr) }()
	s.op.enter()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.op.leave()
		return nil, ErrClosed
	}
	if s.listActive {
		s.mu.Unlock()
		s.op.leave()
		return nil, ErrInUse
	}
	handle := s.native
	s.mu.Unlock()
	if err := s.retryRetainedList(); err != nil {
		s.op.leave()
		return nil, err
	}
	s.mu.Lock()
	s.listActive = true
	s.mu.Unlock()
	request, code := handle.StartList(start, end)
	s.op.leave()
	if code != 0 {
		s.finishList(nil, 0)
		return nil, nativeError(code)
	}
	done := make(chan int32, 1)
	go func() { done <- request.Wait() }()
	var cancelled bool
	select {
	case code = <-done:
	case <-ctx.Done():
		cancelled = true
		_ = request.Cancel()
		code = <-done
	}
	if code == 0 && !cancelled {
		ranges, rangeCode := request.Ranges()
		code = rangeCode
		result = make([]RecordingRange, 0, len(ranges))
		for _, item := range ranges {
			result = append(result, RecordingRange{
				StartTime: time.UnixMilli(item.StartMS).UTC(),
				EndTime:   time.UnixMilli(item.EndMS).UTC(),
			})
		}
	}
	closeCode := request.Close()
	s.finishList(request, closeCode)
	if cancelled {
		return nil, ctx.Err()
	}
	if code != 0 {
		return nil, nativeError(code)
	}
	if closeCode != 0 {
		return nil, nativeError(closeCode)
	}
	if result == nil {
		result = []RecordingRange{}
	}
	return result, nil
}

func (s *CloudStorage) ListRecordingDays(ctx context.Context, startDate, endDate string) ([]RecordingDay, error) {
	return s.ListRecordingDaysInTimeZone(ctx, startDate, endDate, "Asia/Shanghai")
}

func (s *CloudStorage) ListRecordingDaysInTimeZone(
	ctx context.Context,
	startDate string,
	endDate string,
	timeZoneID string,
) (result []RecordingDay, resultErr error) {
	if ctx == nil || startDate == "" || endDate == "" || timeZoneID == "" {
		return nil, ErrInvalidArgument
	}
	logCloudStorageEvent("cloud_storage_list_recording_days_started")
	defer func() { logCloudStorageResult("cloud_storage_list_recording_days", resultErr) }()
	s.op.enter()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.op.leave()
		return nil, ErrClosed
	}
	if s.listActive {
		s.mu.Unlock()
		s.op.leave()
		return nil, ErrInUse
	}
	handle := s.native
	s.mu.Unlock()
	if err := s.retryRetainedList(); err != nil {
		s.op.leave()
		return nil, err
	}
	s.mu.Lock()
	s.listActive = true
	s.mu.Unlock()
	request, code := handle.StartRecordingDays(startDate, endDate, timeZoneID)
	s.op.leave()
	if code != 0 {
		s.finishList(nil, 0)
		return nil, nativeError(code)
	}
	done := make(chan struct {
		days []native.CloudStorageDay
		code int32
	}, 1)
	go func() {
		days, waitCode := request.Wait()
		done <- struct {
			days []native.CloudStorageDay
			code int32
		}{days: days, code: waitCode}
	}()
	var nativeDays []native.CloudStorageDay
	var cancelled bool
	select {
	case terminal := <-done:
		nativeDays = terminal.days
		code = terminal.code
	case <-ctx.Done():
		cancelled = true
		_ = request.Cancel()
		terminal := <-done
		code = terminal.code
	}
	if code == 0 && !cancelled {
		result = make([]RecordingDay, 0, len(nativeDays))
		for _, day := range nativeDays {
			result = append(result, RecordingDay{
				Date:         day.Date,
				HasRecording: day.HasRecording,
			})
		}
	}
	closeCode := request.Close()
	s.finishList(request, closeCode)
	if cancelled {
		return nil, ctx.Err()
	}
	if code != 0 {
		return nil, nativeError(code)
	}
	if closeCode != 0 {
		return nil, nativeError(closeCode)
	}
	if result == nil {
		result = []RecordingDay{}
	}
	return result, nil
}

func (s *CloudStorage) retryRetainedList() error {
	s.mu.Lock()
	request := s.retainedList
	s.mu.Unlock()
	if request == nil {
		return nil
	}
	code := request.Close()
	if code != 0 {
		return nativeError(code)
	}
	s.mu.Lock()
	s.retainedList = nil
	s.mu.Unlock()
	return nil
}

func (s *CloudStorage) finishList(request cloudStorageListReleaser, closeCode int32) {
	s.mu.Lock()
	if closeCode != 0 {
		s.retainedList = request
	}
	s.listActive = false
	s.mu.Unlock()
}

func (s *CloudStorage) Close() (resultErr error) {
	defer func() { logCloudStorageResult("cloud_storage_dispose", resultErr) }()
	s.op.enter()
	if !s.queue.idle() {
		s.op.leave()
		return ErrInUse
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.op.leave()
		return nil
	}
	if s.listActive {
		s.mu.Unlock()
		s.op.leave()
		return ErrInUse
	}
	handle := s.native
	s.mu.Unlock()
	if err := s.retryRetainedList(); err != nil {
		s.op.leave()
		return err
	}
	if err := nativeError(handle.Close()); err != nil {
		s.op.leave()
		return err
	}
	s.mu.Lock()
	s.closed = true
	s.native = nil
	s.mu.Unlock()
	// A native callback may have entered the Go mailbox after the idle check but before
	// the C destroy barrier completed. Mark the handle closed, then release the operation
	// gate before draining that already-admitted user work so callback reentry cannot deadlock.
	finishNativeClose(&s.op, s.queue)
	return nil
}
