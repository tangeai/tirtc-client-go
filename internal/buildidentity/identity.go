package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.6.0-alpha.3"
	buildTimeUTC = "2026-10-22T18:22:18Z"
	sourceRevision = "3e607aa93bb48c47bff3dcd1e16ad20cba01eeb4"
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
