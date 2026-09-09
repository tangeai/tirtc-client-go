package storage

import (
	"github.com/tangeai/tirtc-client-go/v2/internal/native"
	"sync"
	"time"
)

type mediaFileOwner struct {
	mu      sync.Mutex
	path    string
	deleted bool
}

func newRecordingFile(path string, duration time.Duration) RecordingFile {
	return RecordingFile{Path: path, Duration: duration, owner: &mediaFileOwner{path: path}}
}

func deleteMediaFile(path string) error {
	if path == "" {
		return ErrInvalidArgument
	}
	err := nativeError(native.DeleteMediaFile(path))
	logCloudStorageResult("cloud_storage_media_file_delete", err)
	return err
}

// Delete synchronously removes this Runtime-owned temporary media file.
func (f RecordingFile) Delete() error {
	if f.owner == nil {
		return deleteMediaFile(f.Path)
	}
	f.owner.mu.Lock()
	defer f.owner.mu.Unlock()
	if f.Path != f.owner.path {
		return ErrInvalidArgument
	}
	if f.owner.deleted {
		return nil
	}
	if err := deleteMediaFile(f.Path); err != nil {
		return err
	}
	f.owner.deleted = true
	return nil
}

// Delete synchronously removes this Runtime-owned temporary media file.
func (f SnapshotFile) Delete() error { return deleteMediaFile(f.Path) }
