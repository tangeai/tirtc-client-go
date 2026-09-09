package storage

import (
	"context"
	"errors"
	"testing"
)

func TestContextPreservesNativeTerminalWinner(t *testing.T) {
	for _, code := range []int32{0, 6024, 6009, 6114, 6115, 6124} {
		original := nativeError(code)
		got := withCancellationContext(original, context.Canceled)
		cancelled := code == 6115 || code == 6124
		if errors.Is(got, context.Canceled) != cancelled {
			t.Fatalf("code %d: unexpected context classification: %v", code, got)
		}
		if !cancelled && got != original {
			t.Fatalf("code %d: terminal replaced: %v", code, got)
		}
		if original != nil && !errors.Is(got, original) {
			t.Fatalf("code %d: native cause lost: %v", code, got)
		}
	}
}
