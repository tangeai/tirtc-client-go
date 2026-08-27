package tirtc

import (
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
	opMu    sync.Mutex
	mu      sync.Mutex
	native  *native.Conn
	options ConnOptions
	mailbox mailbox
	state   ConnState
	closed  bool
	deps    int
	tasks   int
}

func NewConn(options ConnOptions) (*Conn, error) {
	connection := &Conn{options: options, state: ConnIdle}
	handle, code := native.NewConn(connection.nativeCallbacks())
	if code != 0 {
		err := nativeError(code)
		logSDKResult("connection_create", err)
		return nil, err
	}
	connection.native = handle
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

func (c *Conn) Connect(remoteID, token string) error {
	if remoteID == "" || token == "" {
		return ErrInvalidArgument
	}
	err := c.withNative(func(handle *native.Conn) int32 { return handle.Connect(remoteID, token) })
	logSDKResult("connection_connect", err)
	return err
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
	blocked := c.deps != 0 || c.tasks != 0
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
	defer c.opMu.Unlock()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	if c.deps != 0 || c.tasks != 0 {
		c.mu.Unlock()
		return ErrInUse
	}
	handle := c.native
	state := c.state
	c.mu.Unlock()
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
	return nil
}

func (c *Conn) attachDependency(operation func(*native.Conn) error) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.mu.Lock()
	if c.closed || c.native == nil {
		c.mu.Unlock()
		return ErrClosed
	}
	c.deps++
	handle := c.native
	c.mu.Unlock()
	if err := operation(handle); err != nil {
		c.mu.Lock()
		c.deps--
		c.mu.Unlock()
		return err
	}
	return nil
}

func (c *Conn) detachDependency(operation func(*native.Conn) error) error {
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
	if c.deps > 0 {
		c.deps--
	}
	c.mu.Unlock()
	return nil
}
func validStreamID(id uint8) bool { return id <= 15 }
