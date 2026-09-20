package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.5.0"
	buildTimeUTC = "2026-09-20T13:38:59Z"
	sourceRevision = "b2878b8e185c967a70afed529385e1486890dde8"
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
