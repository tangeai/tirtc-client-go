package tirtc

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

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
	mu          sync.Mutex
	native      *native.Client
	connections map[*Conn]struct{}
	closing     bool
	closed      bool
}

func NewClient(options ClientOptions) (*Client, error) {
	if options.AppID == "" || options.AccessKeyID == "" || options.AccessKeySecret == "" ||
		options.CacheDir == "" || !filepath.IsAbs(options.CacheDir) {
		return nil, ErrInvalidArgument
	}
	cache, err := normalizeDir(options.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrIO, err)
	}
	handle, code := native.NewClient(native.ClientOptions{
		AppID: options.AppID, AccessKeyID: options.AccessKeyID,
		AccessKeySecret: options.AccessKeySecret, Endpoint: options.Endpoint, CacheDir: cache,
		ConsoleLogEnabled: options.ConsoleLogEnabled,
	})
	err = nativeError(code)
	if err == nil {
		logSDKBuildIdentity()
	}
	logSDKResult("client_create", err)
	if err != nil {
		return nil, err
	}
	return &Client{native: handle, connections: make(map[*Conn]struct{})}, nil
}

func (c *Client) Close() error {
	if c == nil {
		return ErrClosed
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	if c.closing {
		c.mu.Unlock()
		return ErrInUse
	}
	c.closing = true
	connections := make([]*Conn, 0, len(c.connections))
	for connection := range c.connections {
		connections = append(connections, connection)
	}
	handle := c.native
	c.mu.Unlock()
	for _, connection := range connections {
		if err := connection.preflightClientClose(); err != nil {
			c.mu.Lock()
			c.closing = false
			c.mu.Unlock()
			return err
		}
	}
	for _, connection := range connections {
		if err := connection.closeFromClient(); err != nil {
			c.mu.Lock()
			c.closing = false
			c.mu.Unlock()
			return err
		}
	}
	err := nativeError(handle.Close())
	c.mu.Lock()
	if err == nil {
		c.native = nil
		c.closed = true
	}
	c.closing = false
	c.mu.Unlock()
	logSDKResult("client_close", err)
	return err
}

func UploadLogs() (string, error) {
	logID, code := native.UploadLogs()
	err := nativeError(code)
	logSDKResult("runtime_upload_logs", err)
	return logID, err
}

func deleteMediaFile(path string) error {
	if path == "" {
		return ErrInvalidArgument
	}
	err := nativeError(native.DeleteMediaFile(path))
	logSDKResult("media_file_delete", err)
	return err
}

func normalizeDir(value string) (string, error) {
	if value == "" || !filepath.IsAbs(value) {
		return "", ErrInvalidArgument
	}
	return filepath.Clean(value), nil
}

func ensureWritableDir(absolute string) error {
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return err
	}
	probe, err := os.CreateTemp(absolute, ".tirtc-write-probe-")
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
