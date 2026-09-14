package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.5.0-alpha.4"
	buildTimeUTC = "2026-09-14T04:04:31Z"
	sourceRevision = "bc69a1d0f5f1feb7c18e32aa200376b993c313b3"
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
