package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.5.0-alpha.1"
	buildTimeUTC = "2026-09-07T04:53:01Z"
	sourceRevision = "ad77038ac4e4f2931435c8dddae9bc9a5c67dd6a"
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
