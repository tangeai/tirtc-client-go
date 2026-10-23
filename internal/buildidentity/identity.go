package buildidentity

import (
	"fmt"
	"sync"
)

const (
	releaseVersion = "2.6.0-alpha.8"
	buildTimeUTC = "2026-10-23T07:53:30Z"
	sourceRevision = "ed3398ab967f681a1e03170afc4f541379243682"
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
