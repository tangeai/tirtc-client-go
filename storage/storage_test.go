package storage

import (
	"testing"
	"time"
)

func TestMultipleCloudStorageClientsHaveIndependentApplicationIdentity(t *testing.T) {
	cache := t.TempDir()
	first, err := NewClient(ClientOptions{AppID: "first", AccessKeyID: "key-1", AccessKeySecret: "secret-1", Endpoint: "first.example", CacheDir: cache})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewClient(ClientOptions{AppID: "second", AccessKeyID: "key-2", AccessKeySecret: "secret-2", Endpoint: "second.example", CacheDir: cache})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClientCloseDetachesBorrowedOutputAndReclaimsReplay(t *testing.T) {
	client, err := NewClient(ClientOptions{AppID: "app", AccessKeyID: "key", AccessKeySecret: "secret", CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := client.NewReplay("device", ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	output, err := NewVideoOutput(VideoOutputOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := output.Attach(replay, 1); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("repeated Client.Close = %v", err)
	}
	if err := replay.Close(); err != nil {
		t.Fatalf("replay was not reclaimed: %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("borrowed output was not left independently owned: %v", err)
	}
}

// A blocked child proves Close takes its snapshot after an admitted creation.
type closePreflightChild struct{}

func (*closePreflightChild) preflightClientClose() error { return ErrInUse }
func (*closePreflightChild) closeFromClient() error      { panic("preflight must prevent teardown") }

func TestClientCloseIncludesAdmittedChildRegistration(t *testing.T) {
	client, err := NewClient(ClientOptions{AppID: "app", AccessKeyID: "key", AccessKeySecret: "secret", CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	client.op.enter()
	done := make(chan error, 1)
	go func() { done <- client.Close() }()
	select {
	case err := <-done:
		client.op.leave()
		t.Fatalf("Close crossed admitted creation: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	child := &closePreflightChild{}
	client.registerChildLocked(child)
	client.op.leave()
	select {
	case err := <-done:
		if err != ErrInUse {
			t.Fatalf("Close omitted registered child's preflight: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not finish")
	}
	client.unregisterChild(child)
	if _, err := client.NewReplay("device", ReplayOptions{}); err != nil {
		t.Fatalf("preflight partially closed client: %v", err)
	}
}
