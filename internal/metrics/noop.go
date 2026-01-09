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

// IncAnalyticsEventPublished is a no-op.
func (n *NoopRecorder) IncAnalyticsEventPublished(status string) {}

// IncAnalyticsEventProcessed is a no-op.
func (n *NoopRecorder) IncAnalyticsEventProcessed(status string) {}

// ObserveAnalyticsBatchSize is a no-op.
func (n *NoopRecorder) ObserveAnalyticsBatchSize(size int) {}

// ObserveAnalyticsBatchDuration is a no-op.
func (n *NoopRecorder) ObserveAnalyticsBatchDuration(duration time.Duration) {}

// SetAnalyticsQueueDepth is a no-op.
func (n *NoopRecorder) SetAnalyticsQueueDepth(depth int64) {}

// ObserveAnalyticsIngestLag is a no-op.
func (n *NoopRecorder) ObserveAnalyticsIngestLag(lag time.Duration) {}

