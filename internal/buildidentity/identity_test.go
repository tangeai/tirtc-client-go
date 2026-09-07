package buildidentity

import (
	"strings"
	"sync"
	"testing"
)

func TestLineIsOncePerProduct(t *testing.T) {
	logged = sync.Map{}
	for _, product := range []string{"rtc", "cloud_storage"} {
		line, ok := Line(product)
		if !ok || !strings.Contains(line, "product="+product) {
			t.Fatalf("missing identity for %s: %q", product, line)
		}
		if _, repeated := Line(product); repeated {
			t.Fatalf("repeated identity for %s", product)
		}
		Release(product)
		if _, retried := Line(product); !retried {
			t.Fatalf("identity did not recover for %s", product)
		}
	}
}
