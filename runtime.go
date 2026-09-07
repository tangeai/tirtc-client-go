package tirtc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	cache, err := normalizeDir(options.CacheDir)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrIO, err)
	}
	options.CacheDir = cache
	err = runtimelease.Init(runtimelease.RTC, runtimelease.Configuration{
		AppID: options.AppID, Endpoint: options.Endpoint, CacheDir: cache,
		ConsoleLogEnabled: options.ConsoleLogEnabled,
	}, func() error {
		if err := ensureWritableDir(cache); err != nil {
			return fmt.Errorf("%w: %w", ErrIO, err)
		}
		return nativeError(native.Init(native.InitOptions{
			AppID: options.AppID, Endpoint: options.Endpoint, CacheDir: cache,
			ConsoleLogEnabled: options.ConsoleLogEnabled,
		}))
	})
	if errors.Is(err, runtimelease.ErrConflict) {
		err = ErrAlreadyInitialized
	}
	logSDKResult("runtime_init", err)
	return err
}

func Shutdown() error {
	logSDKEvent("runtime_shutdown_started")
	err := runtimelease.Shutdown(runtimelease.RTC, func() error { return nativeError(native.Shutdown()) })
	logSDKResult("runtime_shutdown", err)
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
