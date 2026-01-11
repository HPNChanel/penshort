// Package metrics provides lightweight hooks for instrumentation.
package metrics

import "time"

// Recorder captures metric events for the application.
// Implementations can expose these to Prometheus, StatsD, etc.
type Recorder interface {
	// Redirect metrics
	IncRedirectCacheHit()
	IncRedirectCacheMiss()
	ObserveRedirectDuration(duration time.Duration)

	// Link management metrics
	IncLinkCreated()
	IncLinkUpdated()
	IncLinkDeleted()

	// Analytics pipeline metrics
	IncAnalyticsEventPublished(status string) // status: "success" or "dropped"
	IncAnalyticsEventProcessed(status string) // status: "success", "failed", "skipped"
	ObserveAnalyticsBatchSize(size int)
	ObserveAnalyticsBatchDuration(duration time.Duration)
	SetAnalyticsQueueDepth(depth int64)
	ObserveAnalyticsIngestLag(lag time.Duration)

	// Webhook delivery metrics
	IncWebhookDelivery(status string, endpointID string)         // status: "success", "failed", "exhausted"
	ObserveWebhookDeliveryDuration(endpointID string, duration time.Duration)
	IncWebhookRetry(endpointID string, attempt int)
	SetWebhookQueueDepth(depth int64)
}

// Snapshotter exposes a snapshot of current metrics.
type Snapshotter interface {
	Snapshot() Snapshot
}
