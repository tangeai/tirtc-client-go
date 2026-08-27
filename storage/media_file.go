package storage

import "github.com/tangeai/tirtc-client-go/v2/internal/native"

func deleteMediaFile(path string) error {
	if path == "" {
		return ErrInvalidArgument
	}
	err := nativeError(native.DeleteMediaFile(path))
	logCloudStorageResult("cloud_storage_media_file_delete", err)
	return err
}

// Delete synchronously removes this Runtime-owned temporary media file.
func (f RecordingFile) Delete() error { return deleteMediaFile(f.Path) }

// Delete synchronously removes this Runtime-owned temporary media file.
func (f SnapshotFile) Delete() error { return deleteMediaFile(f.Path) }
