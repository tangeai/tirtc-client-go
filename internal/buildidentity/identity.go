package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.6.0-alpha.4"
	buildTimeUTC = "2026-10-22T21:31:10Z"
	sourceRevision = "dc3279ec7771c4cc455cc070da128b4df387da31"
	sourceDirty = 0
)

var logged sync.Map

func Line(product string) (string, bool) {
	if _, loaded := logged.LoadOrStore(product, struct{}{}); loaded {
		return "", false
	}
	return fmt.Sprintf("[build_identity] summary: component build identity; component=go product=%s release_version=%s build_time_utc=%s source_revision=%s source_dirty=%d", product, releaseVersion, buildTimeUTC, sourceRevision, sourceDirty), true
}

func Release(product string) {
	logged.Delete(product)
}
