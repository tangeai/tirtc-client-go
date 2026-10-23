package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.6.0-alpha.13"
	buildTimeUTC = "2026-10-23T19:34:36Z"
	sourceRevision = "68413624485f55c50104e1d670c805ec6e663da4"
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
