package tirtc_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tirtc "github.com/tangeai/tirtc-client-go/v2"
	"github.com/tangeai/tirtc-client-go/v2/storage"
)

func TestRTCAndCloudStorageShareOnlyCoreConfiguration(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "shared")
	if err := tirtc.Init(tirtc.InitOptions{AppID: "rtc-app", Endpoint: "rtc.example", CacheDir: cache}); err != nil {
		t.Fatal(err)
	}
	defer tirtc.Shutdown()
	conflictingCache := filepath.Join(root, "must-not-exist")
	if err := storage.Init(storage.InitOptions{AppID: "cloud-storage-app", Endpoint: "cloud-storage.example", CacheDir: conflictingCache}); !errors.Is(err, storage.ErrAlreadyInitialized) {
		t.Fatalf("conflicting CloudStorage Init = %v", err)
	}
	if _, err := os.Stat(conflictingCache); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("conflicting CloudStorage Init changed filesystem: %v", err)
	}
	if err := storage.Init(storage.InitOptions{AppID: "cloud-storage-app", Endpoint: "cloud-storage.example", CacheDir: cache}); err != nil {
		t.Fatal(err)
	}
	if err := storage.Shutdown(); err != nil {
		t.Fatal(err)
	}
}
