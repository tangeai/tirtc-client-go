package tirtc

import (
	"errors"
	"testing"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type archiveFailureRawDumpNative struct {
	closeCalls int
}

func (*archiveFailureRawDumpNative) Stop() (native.RawDumpArchive, int32) {
	return native.RawDumpArchive{}, 6044
}

func (n *archiveFailureRawDumpNative) Close() int32 {
	n.closeCalls++
	if n.closeCalls == 1 {
		return 6026
	}
	return 0
}

func TestClientCloseReleasesConnectionAfterRawDumpArchiveFailure(t *testing.T) {
	client, err := NewClient(testClientOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	connection, err := client.NewConnection(ConnOptions{})
	if err != nil {
		t.Fatal(err)
	}
	handle := &archiveFailureRawDumpNative{}
	dump := &RawDump{native: handle, parent: connection}
	connection.mu.Lock()
	connection.dumps[dump] = struct{}{}
	connection.mu.Unlock()

	if err := client.Close(); !errors.Is(err, ErrIO) {
		t.Fatalf("Client.Close error = %v", err)
	}
	if handle.closeCalls != 2 {
		t.Fatalf("raw dump destroy calls = %d", handle.closeCalls)
	}
	if !dump.released() || !connection.released() {
		t.Fatal("archive failure retained raw dump or connection")
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
