package metrics

import "time"

// NoopRecorder implements Recorder with no-op methods.
type NoopRecorder struct{}

// NewNoop returns a Recorder that discards all metrics.
func NewNoop() Recorder {
	return &NoopRecorder{}
}

// IncRedirectCacheHit is a no-op.
func (n *NoopRecorder) IncRedirectCacheHit() {}

// IncRedirectCacheMiss is a no-op.
func (n *NoopRecorder) IncRedirectCacheMiss() {}

// ObserveRedirectDuration is a no-op.
func (n *NoopRecorder) ObserveRedirectDuration(duration time.Duration) {}

// IncLinkCreated is a no-op.
func (n *NoopRecorder) IncLinkCreated() {}

// IncLinkUpdated is a no-op.
func (n *NoopRecorder) IncLinkUpdated() {}

// IncLinkDeleted is a no-op.
func (n *NoopRecorder) IncLinkDeleted() {}
