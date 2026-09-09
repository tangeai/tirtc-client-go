package storage

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type ClientOptions struct {
	AppID             string
	AccessKeyID       string
	AccessKeySecret   string
	CacheDir          string
	Endpoint          string
	ConsoleLogEnabled bool
}

type Client struct {
	op       nativeOperationGate
	mu       sync.Mutex
	native   *native.CloudStorageClient
	queue    *callbackQueue
	children map[clientChild]struct{}
	closing  bool
	closed   bool
}

type clientChild interface {
	preflightClientClose() error
	closeFromClient() error
}

type clientCall struct {
	cancel func() int32
	done   chan struct{}
}

func (o *clientCall) preflightClientClose() error { return nil }
func (o *clientCall) closeFromClient() error {
	if o.cancel != nil {
		_ = o.cancel()
	}
	<-o.done
	return nil
}

func NewClient(options ClientOptions) (*Client, error) {
	if options.AppID == "" || options.AccessKeyID == "" || options.AccessKeySecret == "" ||
		options.CacheDir == "" || !filepath.IsAbs(options.CacheDir) {
		return nil, ErrInvalidArgument
	}
	options.CacheDir = filepath.Clean(options.CacheDir)
	client := &Client{queue: newCallbackQueue(), children: make(map[clientChild]struct{})}
	handle, code := native.NewCloudStorageClient(native.ClientOptions{
		AppID: options.AppID, AccessKeyID: options.AccessKeyID,
		AccessKeySecret: options.AccessKeySecret, Endpoint: options.Endpoint,
		CacheDir: options.CacheDir, ConsoleLogEnabled: options.ConsoleLogEnabled,
	})
	if code != 0 {
		client.queue.close()
		err := nativeError(code)
		logCloudStorageResult("cloud_storage_client_create", err)
		return nil, err
	}
	client.native = handle
	logCloudStorageBuildIdentity()
	logCloudStorageResult("cloud_storage_client_create", nil)
	return client, nil
}

// openDeviceLocked is called while c.op is held so the device and its language
// child can be registered before Client.Close starts its cleanup barrier.
func (c *Client) openDeviceLocked(deviceID string) (*native.CloudStorage, error) {
	if c == nil || deviceID == "" {
		return nil, ErrInvalidArgument
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.closing || c.native == nil {
		return nil, ErrClosed
	}
	device, code := c.native.OpenDevice(deviceID)
	if code != 0 {
		return nil, nativeError(code)
	}
	return device, nil
}

func (c *Client) closeDevice(device *native.CloudStorage) error {
	if device == nil {
		return nil
	}
	err := nativeError(device.Close())
	return err
}

func (c *Client) registerChildLocked(child clientChild) {
	c.mu.Lock()
	c.children[child] = struct{}{}
	c.mu.Unlock()
}

func (c *Client) unregisterChild(child clientChild) {
	c.mu.Lock()
	delete(c.children, child)
	c.mu.Unlock()
}

func (c *Client) ListRecordings(ctx context.Context, deviceID string, startTime, endTime time.Time) (result []RecordingRange, resultErr error) {
	if ctx == nil || deviceID == "" {
		return nil, ErrInvalidArgument
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	start := unixMilliseconds(startTime)
	end := unixMilliseconds(endTime)
	if start < 0 || start >= end {
		return nil, ErrInvalidArgument
	}
	logCloudStorageEvent("cloud_storage_list_recordings_started")
	defer func() { logCloudStorageResult("cloud_storage_list_recordings", resultErr) }()
	c.op.enter()
	device, err := c.openDeviceLocked(deviceID)
	if err != nil {
		c.op.leave()
		return nil, err
	}
	request, code := device.StartList(start, end)
	if code != 0 {
		_ = c.closeDevice(device)
		c.op.leave()
		return nil, nativeError(code)
	}
	call := &clientCall{cancel: request.Cancel, done: make(chan struct{})}
	c.registerChildLocked(call)
	c.op.leave()
	defer func() {
		c.unregisterChild(call)
		close(call.done)
	}()
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
	if code == 0 {
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
	deviceCloseErr := c.closeDevice(device)
	if code != 0 {
		err := nativeError(code)
		if cancelled {
			err = withCancellationContext(err, ctx.Err())
		}
		return nil, err
	}
	if closeCode != 0 {
		return nil, nativeError(closeCode)
	}
	if deviceCloseErr != nil {
		return nil, deviceCloseErr
	}
	if result == nil {
		result = []RecordingRange{}
	}
	return result, nil
}

func (c *Client) ListRecordingDays(ctx context.Context, deviceID, startDate, endDate string) ([]RecordingDay, error) {
	return c.ListRecordingDaysInTimeZone(ctx, deviceID, startDate, endDate, "Asia/Shanghai")
}

func (c *Client) ListRecordingDaysInTimeZone(
	ctx context.Context,
	deviceID string,
	startDate string,
	endDate string,
	timeZoneID string,
) (result []RecordingDay, resultErr error) {
	if ctx == nil || deviceID == "" || startDate == "" || endDate == "" || timeZoneID == "" {
		return nil, ErrInvalidArgument
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	logCloudStorageEvent("cloud_storage_list_recording_days_started")
	defer func() { logCloudStorageResult("cloud_storage_list_recording_days", resultErr) }()
	c.op.enter()
	device, err := c.openDeviceLocked(deviceID)
	if err != nil {
		c.op.leave()
		return nil, err
	}
	request, code := device.StartRecordingDays(startDate, endDate, timeZoneID)
	if code != 0 {
		_ = c.closeDevice(device)
		c.op.leave()
		return nil, nativeError(code)
	}
	call := &clientCall{cancel: request.Cancel, done: make(chan struct{})}
	c.registerChildLocked(call)
	c.op.leave()
	defer func() {
		c.unregisterChild(call)
		close(call.done)
	}()
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
		nativeDays = terminal.days
		code = terminal.code
	}
	if code == 0 {
		result = make([]RecordingDay, 0, len(nativeDays))
		for _, day := range nativeDays {
			result = append(result, RecordingDay{
				Date:         day.Date,
				HasRecording: day.HasRecording,
			})
		}
	}
	closeCode := request.Close()
	deviceCloseErr := c.closeDevice(device)
	if code != 0 {
		err := nativeError(code)
		if cancelled {
			err = withCancellationContext(err, ctx.Err())
		}
		return nil, err
	}
	if closeCode != 0 {
		return nil, nativeError(closeCode)
	}
	if deviceCloseErr != nil {
		return nil, deviceCloseErr
	}
	if result == nil {
		result = []RecordingDay{}
	}
	return result, nil
}

func (c *Client) Close() (resultErr error) {
	defer func() { logCloudStorageResult("cloud_storage_dispose", resultErr) }()
	if c == nil {
		return ErrClosed
	}
	if !c.queue.idle() {
		return ErrInUse
	}
	// Finish admitted child creation before taking the managed-close snapshot.
	c.op.enter()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		c.op.leave()
		return nil
	}
	if c.closing {
		c.mu.Unlock()
		c.op.leave()
		return ErrInUse
	}
	c.closing = true
	children := make([]clientChild, 0, len(c.children))
	for child := range c.children {
		children = append(children, child)
	}
	handle := c.native
	c.mu.Unlock()
	c.op.leave()
	for _, child := range children {
		if err := child.preflightClientClose(); err != nil {
			c.mu.Lock()
			c.closing = false
			c.mu.Unlock()
			return err
		}
	}
	for _, child := range children {
		if err := child.closeFromClient(); err != nil {
			c.mu.Lock()
			c.closing = false
			c.mu.Unlock()
			return err
		}
	}
	c.op.enter()
	if err := nativeError(handle.Close()); err != nil {
		c.op.leave()
		c.mu.Lock()
		c.closing = false
		c.mu.Unlock()
		return err
	}
	c.mu.Lock()
	c.closed = true
	c.closing = false
	c.native = nil
	c.mu.Unlock()
	// A native callback may have entered the Go mailbox after the idle check but before
	// the C destroy barrier completed. Mark the handle closed, then release the operation
	// gate before draining that already-admitted user work so callback reentry cannot deadlock.
	finishNativeClose(&c.op, c.queue)
	return nil
}
