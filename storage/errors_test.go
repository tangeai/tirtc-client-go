package storage

import (
	"errors"
	"testing"
)

func TestConnectivityErrorsPreserveStableSentinel(t *testing.T) {
	for code, sentinel := range map[int32]error{
		6136: ErrNetworkUnavailable,
		6137: ErrEndpointDNSResolutionFailed,
	} {
		if err := nativeError(code); !errors.Is(err, sentinel) {
			t.Fatalf("errors.Is(%d, %v) = false: %v", code, sentinel, err)
		}
	}
}
