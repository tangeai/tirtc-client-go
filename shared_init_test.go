package tirtc_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tirtc "github.com/tangeai/tirtc-client-go/v2"
	"github.com/tangeai/tirtc-client-go/v2/storage"
)

func TestRTCAndCloudStorageShareOnlyHostConfiguration(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "shared")
	rtc, err := tirtc.NewClient(tirtc.ClientOptions{AppID: "rtc-app", AccessKeyID: "key", AccessKeySecret: "secret", Endpoint: "rtc.example", CacheDir: cache})
	if err != nil {
		t.Fatal(err)
	}
	defer rtc.Close()
	conflicting := filepath.Join(root, "must-not-exist")
	if _, err := storage.NewClient(storage.ClientOptions{AppID: "storage-app", AccessKeyID: "key", AccessKeySecret: "secret", CacheDir: conflicting}); !errors.Is(err, storage.ErrAlreadyInitialized) {
		t.Fatalf("conflicting storage Client = %v", err)
	}
	if _, err := os.Stat(conflicting); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("conflicting Client changed filesystem: %v", err)
	}
	cloud, err := storage.NewClient(storage.ClientOptions{AppID: "storage-app", AccessKeyID: "key", AccessKeySecret: "secret", Endpoint: "cloud.example", CacheDir: cache})
	if err != nil {
		t.Fatal(err)
	}
	if err := cloud.Close(); err != nil {
		t.Fatal(err)
	}
}
