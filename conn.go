package tirtc

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

const firstApplicationCommandID uint32 = 0x2001

type ConnOptions struct {
	OnStateChanged  func(state ConnState, err error)
	OnCommand       func(commandID uint32, data []byte)
	OnStreamMessage func(streamID uint8, timestamp time.Duration, data []byte)
}

type Conn struct {
	opMu        sync.Mutex
	mu          sync.Mutex
	native      *native.Conn
	client      *Client
	options     ConnOptions
	mailbox     mailbox
	state       ConnState
	closed      bool
	deps        map[connectionDependency]struct{}
	tasks       map[*RecordingTask]struct{}
	connectDone chan struct{}
}

type connectionDependency interface {
	preflightClientClose() error
	detachFromClient() error
}

func (c *Client) NewConnection(options ConnOptions) (*Conn, error) {
	if c == nil {
		return nil, ErrClosed
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.native == nil {
		return nil, ErrClosed
	}
	if c.closing {
		return nil, ErrClosed
	}
	connection := &Conn{client: c, options: options, state: ConnIdle,
		deps: make(map[connectionDependency]struct{}), tasks: make(map[*RecordingTask]struct{})}
	handle, code := c.native.NewConn(connection.nativeCallbacks())
	if code != 0 {
		err := nativeError(code)
		logSDKResult("connection_create", err)
		return nil, err
	}
	connection.native = handle
	c.connections[connection] = struct{}{}
	logSDKResult("connection_create", nil)
	return connection, nil
}

func (c *Conn) nativeCallbacks() native.ConnCallbacks {
	return native.ConnCallbacks{
		OnState: func(state uint32, code int32) {
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			changed := c.state != ConnState(state)
			c.state = ConnState(state)
			c.mu.Unlock()
			err := nativeError(code)
			if changed || err != nil {
				logSDKState("connection_state", state, err)
			}
			if c.options.OnStateChanged == nil || (!changed && err == nil) {
				return
			}
			_ = c.mailbox.postLatest(mailboxLatestState, func() {
				c.options.OnStateChanged(ConnState(state), nativeError(code))
			})
		},
		OnCommand: func(command uint32, data []byte) {
			if c.options.OnCommand == nil {
				return
			}
			if !c.mailbox.postData(func() {
				c.options.OnCommand(command, data)
			}) {
				_ = c.mailbox.postStopRequest(func() { _ = c.Disconnect() })
			}
		},
		OnMessage: func(stream uint8, timestamp uint32, data []byte) {
			if c.options.OnStreamMessage == nil {
				return
			}
			if !c.mailbox.postData(func() {
				c.options.OnStreamMessage(stream, time.Duration(timestamp)*time.Millisecond, data)
			}) {
				_ = c.mailbox.postStopRequest(func() { _ = c.Disconnect() })
			}
		},
	}
}

func (c *Conn) State() ConnState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *Conn) withNative(operation func(*native.Conn) int32) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.mu.Lock()
	if c.closed || c.native == nil {
		c.mu.Unlock()
		return ErrClosed
	}
	handle := c.native
	c.mu.Unlock()
	return nativeError(operation(handle))
}

func (c *Conn) Connect(ctx context.Context, deviceID string) error {
	if ctx == nil || deviceID == "" {
		return ErrInvalidArgument
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	timeout := time.Duration(0)
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			return ctx.Err()
		}
		if timeout < time.Millisecond {
			timeout = time.Millisecond
		}
		if timeout > 120*time.Second {
			timeout = 120 * time.Second
		}
	}
	c.opMu.Lock()
	c.mu.Lock()
	if c.closed || c.native == nil {
		c.mu.Unlock()
		c.opMu.Unlock()
		return ErrClosed
	}
	if c.connectDone != nil {
		c.mu.Unlock()
		c.opMu.Unlock()
		return ErrInUse
	}
	handle := c.native
	attempt, code := handle.ConnectManaged(deviceID, timeout)
	if code != 0 {
		c.mu.Unlock()
		c.opMu.Unlock()
		err := nativeError(code)
		logSDKResult("connection_connect", err)
		return err
	}
	finished := make(chan struct{})
	c.connectDone = finished
	c.mu.Unlock()
	c.opMu.Unlock()
	cancelledByContext := false
	select {
	case code = <-attempt.Done():
	case <-ctx.Done():
		cancelledByContext = true
		_ = attempt.Cancel()
		code = <-attempt.Done()
	}
	closeCode := attempt.Close()
	if errors.Is(nativeError(closeCode), ErrInUse) {
		go reapConnectAttempt(attempt)
		closeCode = 0
	}
	c.mu.Lock()
	if c.connectDone == finished {
		c.connectDone = nil
		close(finished)
	}
	c.mu.Unlock()
	err := nativeError(code)
	if err == nil {
		err = nativeError(closeCode)
	}
	if cancelledByContext && code != 0 {
		err = withCancellationContext(err, ctx.Err())
	}
	logSDKResult("connection_connect", err)
	return err
}

func reapConnectAttempt(attempt *native.ConnectAttempt) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := nativeError(attempt.Close())
		if !errors.Is(err, ErrInUse) {
			logSDKResult("connection_connect_attempt_cleanup", err)
			return
		}
		<-ticker.C
	}
}

func (c *Conn) Disconnect() error {
	err := c.withNative(func(handle *native.Conn) int32 { return handle.Disconnect() })
	logSDKResult("connection_disconnect", err)
	return err
}

func (c *Conn) SendCommand(commandID uint32, data []byte) error {
	if commandID < firstApplicationCommandID {
		return ErrInvalidArgument
	}
	return c.withNative(func(handle *native.Conn) int32 { return handle.SendCommand(commandID, data) })
}

func (c *Conn) SendStreamMessage(streamID uint8, timestamp time.Duration, data []byte) error {
	if !validStreamID(streamID) || timestamp < 0 || timestamp%time.Millisecond != 0 ||
		timestamp/time.Millisecond > time.Duration(^uint32(0)) {
		return ErrInvalidArgument
	}
	return c.withNative(func(handle *native.Conn) int32 {
		return handle.SendMessage(streamID, uint32(timestamp/time.Millisecond), data)
	})
}

func (c *Conn) SubscribeAudio(streamID uint8) error {
	if !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	return c.withNative(func(handle *native.Conn) int32 { return handle.SubscribeAudio(streamID) })
}
func (c *Conn) UnsubscribeAudio(streamID uint8) error {
	if !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	return c.withNative(func(handle *native.Conn) int32 { return handle.UnsubscribeAudio(streamID) })
}
func (c *Conn) SubscribeVideo(streamID uint8) error {
	if !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	return c.withNative(func(handle *native.Conn) int32 { return handle.SubscribeVideo(streamID) })
}
func (c *Conn) UnsubscribeVideo(streamID uint8) error {
	if !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	return c.withNative(func(handle *native.Conn) int32 { return handle.UnsubscribeVideo(streamID) })
}
func (c *Conn) RequestVideoKeyframe(streamID uint8) error {
	if !validStreamID(streamID) {
		return ErrInvalidArgument
	}
	return c.withNative(func(handle *native.Conn) int32 { return handle.RequestVideoKeyframe(streamID) })
}

func (c *Conn) Close() (resultErr error) {
	defer func() { logSDKResult("connection_dispose", resultErr) }()
	c.mu.Lock()
	alreadyClosed := c.closed
	blocked := len(c.deps) != 0 || len(c.tasks) != 0
	c.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	if blocked {
		return ErrInUse
	}
	if err := c.mailbox.preflightClose(); err != nil {
		return err
	}
	c.opMu.Lock()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		c.opMu.Unlock()
		return nil
	}
	if len(c.deps) != 0 || len(c.tasks) != 0 {
		c.mu.Unlock()
		c.opMu.Unlock()
		return ErrInUse
	}
	handle := c.native
	state := c.state
	connectDone := c.connectDone
	c.mu.Unlock()
	if connectDone != nil {
		if err := nativeError(handle.Disconnect()); err != nil {
			c.opMu.Unlock()
			return err
		}
		c.opMu.Unlock()
		<-connectDone
		c.opMu.Lock()
		c.mu.Lock()
		if c.closed || c.native == nil {
			c.mu.Unlock()
			c.opMu.Unlock()
			return nil
		}
		handle = c.native
		state = c.state
		c.mu.Unlock()
	}
	defer c.opMu.Unlock()
	if err := c.mailbox.beginClose(); err != nil {
		return err
	}
	if state == ConnConnecting || state == ConnConnected {
		if err := nativeError(handle.Disconnect()); err != nil {
			c.mailbox.cancelClose()
			return err
		}
	}
	if err := nativeError(handle.Close()); err != nil {
		c.mailbox.cancelClose()
		return err
	}
	c.mu.Lock()
	c.closed = true
	c.native = nil
	c.state = ConnDisconnected
	c.mu.Unlock()
	c.mailbox.finishClose()
	if c.client != nil {
		c.client.mu.Lock()
		delete(c.client.connections, c)
		c.client.mu.Unlock()
		c.client = nil
	}
	return nil
}

func (c *Conn) attachDependency(dependency connectionDependency,
	operation func(*native.Conn) error) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.mu.Lock()
	if c.closed || c.native == nil {
		c.mu.Unlock()
		return ErrClosed
	}
	handle := c.native
	c.mu.Unlock()
	if err := operation(handle); err != nil {
		return err
	}
	c.mu.Lock()
	c.deps[dependency] = struct{}{}
	c.mu.Unlock()
	return nil
}

func (c *Conn) detachDependency(dependency connectionDependency,
	operation func(*native.Conn) error) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.mu.Lock()
	if c.closed || c.native == nil {
		c.mu.Unlock()
		return ErrClosed
	}
	handle := c.native
	c.mu.Unlock()
	if err := operation(handle); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.deps, dependency)
	c.mu.Unlock()
	return nil
}

func (c *Conn) preflightClientClose() error {
	if err := c.mailbox.preflightClose(); err != nil {
		return err
	}
	c.mu.Lock()
	dependencies := make([]connectionDependency, 0, len(c.deps))
	for dependency := range c.deps {
		dependencies = append(dependencies, dependency)
	}
	c.mu.Unlock()
	for _, dependency := range dependencies {
		if err := dependency.preflightClientClose(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) closeFromClient() error {
	c.mu.Lock()
	dependencies := make([]connectionDependency, 0, len(c.deps))
	for dependency := range c.deps {
		dependencies = append(dependencies, dependency)
	}
	tasks := make([]*RecordingTask, 0, len(c.tasks))
	for task := range c.tasks {
		tasks = append(tasks, task)
	}
	c.mu.Unlock()
	for _, task := range tasks {
		if _, err := task.Stop(); err != nil {
			task.mu.Lock()
			finished := task.finished
			task.mu.Unlock()
			if !finished {
				return err
			}
		}
	}
	for _, dependency := range dependencies {
		if err := dependency.detachFromClient(); err != nil {
			return err
		}
	}
	return c.Close()
}
func validStreamID(id uint8) bool { return id <= 15 }
