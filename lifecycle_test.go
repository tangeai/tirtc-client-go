package tirtc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tangeai/tirtc-client-go/v2/internal/buildidentity"
)

func testClientOptions(cache string) ClientOptions {
	return ClientOptions{AppID: "app", AccessKeyID: "key", AccessKeySecret: "secret", CacheDir: cache}
}

func TestPublicClientConsumesRTCBuildIdentity(t *testing.T) {
	buildidentity.Release("rtc")
	client, err := NewClient(testClientOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, available := buildidentity.Line("rtc"); available {
		t.Fatal("RTC Client did not consume the build identity")
	}
}

func TestSecondRTCClientIsRejectedBeforeFilesystemWrites(t *testing.T) {
	root := t.TempDir()
	client, err := NewClient(testClientOptions(filepath.Join(root, "active")))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	conflicting := filepath.Join(root, "must-not-exist", "cache")
	if _, err := NewClient(testClientOptions(conflicting)); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("second Client = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(conflicting)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("second Client changed filesystem: %v", err)
	}
}

func TestClientRejectsMissingCredentialsBeforeFilesystemWrites(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "missing")
	if _, err := NewClient(ClientOptions{AppID: "app", CacheDir: cache}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("Client = %v", err)
	}
	if _, err := os.Stat(cache); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid Client changed filesystem: %v", err)
	}
}

func TestConnectionClosePreflightsOutputBinding(t *testing.T) {
	client, err := NewClient(testClientOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	connection, err := client.NewConnection(ConnOptions{})
	if err != nil {
		t.Fatal(err)
	}
	output, err := NewVideoOutput(VideoOutputOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := output.Attach(connection, 11); err != nil {
		t.Fatal(err)
	}
	if err := connection.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("connection Close = %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClientCloseDetachesBorrowedOutputAndReclaimsConnection(t *testing.T) {
	client, err := NewClient(testClientOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	connection, err := client.NewConnection(ConnOptions{})
	if err != nil {
		t.Fatal(err)
	}
	output, err := NewVideoOutput(VideoOutputOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := output.Attach(connection, 11); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("repeated Client.Close = %v", err)
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("connection was not reclaimed: %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("borrowed output was not left independently owned: %v", err)
	}
}

func TestRequiredOutputFrameHandlers(t *testing.T) {
	if _, err := NewAudioOutput(AudioOutputOptions{}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("audio output = %v", err)
	}
	if _, err := NewEncodedAudioOutput(EncodedAudioOutputOptions{}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("encoded audio output = %v", err)
	}
	if _, err := NewEncodedVideoOutput(EncodedVideoOutputOptions{}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("encoded video output = %v", err)
	}
}
