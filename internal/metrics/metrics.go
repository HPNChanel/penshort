// Package metrics provides lightweight hooks for instrumentation.
package metrics

import "time"

// Recorder captures metric events for the application.
// Implementations can expose these to Prometheus, StatsD, etc.
type Recorder interface {
	IncRedirectCacheHit()
	IncRedirectCacheMiss()
	ObserveRedirectDuration(duration time.Duration)
	IncLinkCreated()
	IncLinkUpdated()
	IncLinkDeleted()
}
