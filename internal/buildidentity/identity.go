package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.5.0-alpha.2"
	buildTimeUTC = "2026-09-09T19:21:53Z"
	sourceRevision = "4b328129c3be18eb549ac67a77d99609937b4028"
	sourceDirty = 1
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
