package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.6.0-alpha.2"
	buildTimeUTC = "2026-10-22T05:51:00Z"
	sourceRevision = "0a0334e908fa19ae34540577174cd4501e7de741"
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
