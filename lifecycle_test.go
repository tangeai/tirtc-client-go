package tirtc

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestRepeatedInitChecksConfigurationBeforeFilesystemWrites(t *testing.T) {
	root := t.TempDir()
	options := InitOptions{AppID: "app", CacheDir: root + "/runtime"}
	if err := Init(options); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := Shutdown(); err != nil {
			t.Error(err)
		}
	}()
	if err := Init(options); err != nil {
		t.Fatalf("repeated identical Init = %v", err)
	}
	conflictingCache := filepath.Join(root, "must-not-exist", "cache")
	if err := Init(InitOptions{
		AppID: "other", CacheDir: conflictingCache,
	}); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("conflicting Init = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-exist")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("conflicting Init changed filesystem: %v", err)
	}
}

func TestInitRejectsEmptyAppIDBeforeFilesystemWrites(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "runtime")
	if err := Init(InitOptions{CacheDir: cache}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("Init = %v", err)
	}
	if _, err := os.Stat(cache); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid Init changed filesystem: %v", err)
	}
}

func TestInitPreservesPathErrorForFilesystemFailure(t *testing.T) {
	root := t.TempDir()
	blockedParent := filepath.Join(root, "file")
	if err := os.WriteFile(blockedParent, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Init(InitOptions{AppID: "app", CacheDir: filepath.Join(blockedParent, "cache")})
	if !errors.Is(err, ErrIO) {
		t.Fatalf("Init does not preserve ErrIO: %v", err)
	}
	var pathError *os.PathError
	if !errors.As(err, &pathError) {
		t.Fatalf("Init does not preserve *os.PathError: %T %v", err, err)
	}
}

func TestConnectionClosePreflightsOutputBinding(t *testing.T) {
	root := t.TempDir()
	if err := Init(InitOptions{AppID: "app", CacheDir: root + "/runtime"}); err != nil {
		t.Fatal(err)
	}
	connection, err := NewConn(ConnOptions{})
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
	if err := Shutdown(); err != nil {
		t.Fatal(err)
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

func TestConcurrentAttachAndConnectionCloseAreSerialized(t *testing.T) {
	root := t.TempDir()
	if err := Init(InitOptions{AppID: "app", CacheDir: root + "/runtime"}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := Shutdown(); err != nil {
			t.Error(err)
		}
	}()

	for iteration := 0; iteration < 64; iteration++ {
		connection, err := NewConn(ConnOptions{})
		if err != nil {
			t.Fatal(err)
		}
		output, err := NewVideoOutput(VideoOutputOptions{})
		if err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		var wait sync.WaitGroup
		var attachErr, closeErr error
		wait.Add(2)
		go func() {
			defer wait.Done()
			<-start
			attachErr = output.Attach(connection, 11)
		}()
		go func() {
			defer wait.Done()
			<-start
			closeErr = connection.Close()
		}()
		close(start)
		wait.Wait()

		if attachErr == nil {
			if !errors.Is(closeErr, ErrInUse) {
				t.Fatalf("iteration %d: close after attach = %v", iteration, closeErr)
			}
		} else {
			if !errors.Is(attachErr, ErrClosed) {
				t.Fatalf("iteration %d: attach = %v", iteration, attachErr)
			}
			if closeErr != nil {
				t.Fatalf("iteration %d: winning close = %v", iteration, closeErr)
			}
		}
		if err := output.Close(); err != nil {
			t.Fatalf("iteration %d: output close = %v", iteration, err)
		}
		if err := connection.Close(); err != nil {
			t.Fatalf("iteration %d: final connection close = %v", iteration, err)
		}
	}
}
