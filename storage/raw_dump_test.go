package storage

import (
	"errors"
	"testing"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type archiveFailureRawDumpNative struct {
	closeCalls int
}

func (*archiveFailureRawDumpNative) Stop() (native.RawDumpArchive, int32) {
	return native.RawDumpArchive{}, 6046
}

func (n *archiveFailureRawDumpNative) Close() int32 {
	n.closeCalls++
	if n.closeCalls == 1 {
		return 6026
	}
	return 0
}

func TestClientCloseReleasesReplayAfterRawDumpArchiveFailure(t *testing.T) {
	client, err := NewClient(ClientOptions{
		AppID: "app", AccessKeyID: "key", AccessKeySecret: "secret", CacheDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := client.NewReplay("device", ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	handle := &archiveFailureRawDumpNative{}
	dump := &RawDump{native: handle, parent: replay}
	replay.mu.Lock()
	replay.dumps[dump] = struct{}{}
	replay.mu.Unlock()

	if err := client.Close(); !errors.Is(err, ErrIO) {
		t.Fatalf("Client.Close error = %v", err)
	}
	if handle.closeCalls != 2 {
		t.Fatalf("raw dump destroy calls = %d", handle.closeCalls)
	}
	if !dump.released() || !replay.released() {
		t.Fatal("archive failure retained raw dump or replay")
	}
	client.mu.Lock()
	closed := client.closed
	client.mu.Unlock()
	if !closed {
		t.Fatal("archive failure retained client")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("repeated Client.Close = %v", err)
	}
}
