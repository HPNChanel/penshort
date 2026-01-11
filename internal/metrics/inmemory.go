package metrics

import (
	"sync/atomic"
	"time"
)

// Snapshot captures current in-memory counters.
type Snapshot struct {
	RedirectCacheHits       uint64
	RedirectCacheMisses     uint64
	RedirectDurationCount   uint64
	RedirectDurationTotalNs int64
	LinksCreated            uint64
	LinksUpdated            uint64
	LinksDeleted            uint64
	// Analytics metrics
	AnalyticsEventsPublished        uint64
	AnalyticsEventsDropped          uint64
	AnalyticsEventsProcessed        uint64
	AnalyticsEventsProcessedFailed  uint64
	AnalyticsEventsProcessedSkipped uint64
	AnalyticsBatchCount             uint64
	AnalyticsQueueDepth             int64
	AnalyticsBatchDurationCount     uint64
	AnalyticsBatchDurationTotalNs   int64
	AnalyticsIngestLagCount         uint64
	AnalyticsIngestLagTotalNs       int64
	// Webhook metrics
	WebhookDeliveriesSuccess   uint64
	WebhookDeliveriesFailed    uint64
	WebhookDeliveriesExhausted uint64
	WebhookRetries             uint64
	WebhookQueueDepth          int64
	WebhookDurationCount       uint64
	WebhookDurationTotalNs     int64
}

// InMemoryRecorder stores metrics in memory for tests.
type InMemoryRecorder struct {
	redirectCacheHits       uint64
	redirectCacheMisses     uint64
	redirectDurationCount   uint64
	redirectDurationTotalNs int64
	linksCreated            uint64
	linksUpdated            uint64
	linksDeleted            uint64
	// Analytics fields
	analyticsEventsPublished      uint64
	analyticsEventsDropped        uint64
	analyticsEventsProcessed      uint64
	analyticsEventsFailed         uint64
	analyticsEventsSkipped        uint64
	analyticsBatchCount           uint64
	analyticsQueueDepth           int64
	analyticsBatchDurationCount   uint64
	analyticsBatchDurationTotalNs int64
	analyticsIngestLagCount       uint64
	analyticsIngestLagTotalNs     int64
	// Webhook fields
	webhookDeliveriesSuccess   uint64
	webhookDeliveriesFailed    uint64
	webhookDeliveriesExhausted uint64
	webhookRetries             uint64
	webhookQueueDepth          int64
	webhookDurationCount       uint64
	webhookDurationTotalNs     int64
}

// NewInMemory returns a Recorder that stores counters in memory.
func NewInMemory() *InMemoryRecorder {
	return &InMemoryRecorder{}
}

// Snapshot returns a copy of the counters.
func (m *InMemoryRecorder) Snapshot() Snapshot {
	return Snapshot{
		RedirectCacheHits:               atomic.LoadUint64(&m.redirectCacheHits),
		RedirectCacheMisses:             atomic.LoadUint64(&m.redirectCacheMisses),
		RedirectDurationCount:           atomic.LoadUint64(&m.redirectDurationCount),
		RedirectDurationTotalNs:         atomic.LoadInt64(&m.redirectDurationTotalNs),
		LinksCreated:                    atomic.LoadUint64(&m.linksCreated),
		LinksUpdated:                    atomic.LoadUint64(&m.linksUpdated),
		LinksDeleted:                    atomic.LoadUint64(&m.linksDeleted),
		AnalyticsEventsPublished:        atomic.LoadUint64(&m.analyticsEventsPublished),
		AnalyticsEventsDropped:          atomic.LoadUint64(&m.analyticsEventsDropped),
		AnalyticsEventsProcessed:        atomic.LoadUint64(&m.analyticsEventsProcessed),
		AnalyticsEventsProcessedFailed:  atomic.LoadUint64(&m.analyticsEventsFailed),
		AnalyticsEventsProcessedSkipped: atomic.LoadUint64(&m.analyticsEventsSkipped),
		AnalyticsBatchCount:             atomic.LoadUint64(&m.analyticsBatchCount),
		AnalyticsQueueDepth:             atomic.LoadInt64(&m.analyticsQueueDepth),
		AnalyticsBatchDurationCount:     atomic.LoadUint64(&m.analyticsBatchDurationCount),
		AnalyticsBatchDurationTotalNs:   atomic.LoadInt64(&m.analyticsBatchDurationTotalNs),
		AnalyticsIngestLagCount:         atomic.LoadUint64(&m.analyticsIngestLagCount),
		AnalyticsIngestLagTotalNs:       atomic.LoadInt64(&m.analyticsIngestLagTotalNs),
		// Webhook metrics
		WebhookDeliveriesSuccess:   atomic.LoadUint64(&m.webhookDeliveriesSuccess),
		WebhookDeliveriesFailed:    atomic.LoadUint64(&m.webhookDeliveriesFailed),
		WebhookDeliveriesExhausted: atomic.LoadUint64(&m.webhookDeliveriesExhausted),
		WebhookRetries:             atomic.LoadUint64(&m.webhookRetries),
		WebhookQueueDepth:          atomic.LoadInt64(&m.webhookQueueDepth),
		WebhookDurationCount:       atomic.LoadUint64(&m.webhookDurationCount),
		WebhookDurationTotalNs:     atomic.LoadInt64(&m.webhookDurationTotalNs),
	}
}

// IncRedirectCacheHit increments cache hit counter.
func (m *InMemoryRecorder) IncRedirectCacheHit() {
	atomic.AddUint64(&m.redirectCacheHits, 1)
}

// IncRedirectCacheMiss increments cache miss counter.
func (m *InMemoryRecorder) IncRedirectCacheMiss() {
	atomic.AddUint64(&m.redirectCacheMisses, 1)
}

// ObserveRedirectDuration records redirect duration.
func (m *InMemoryRecorder) ObserveRedirectDuration(duration time.Duration) {
	atomic.AddUint64(&m.redirectDurationCount, 1)
	atomic.AddInt64(&m.redirectDurationTotalNs, duration.Nanoseconds())
}

// IncLinkCreated increments link created counter.
func (m *InMemoryRecorder) IncLinkCreated() {
	atomic.AddUint64(&m.linksCreated, 1)
}

// IncLinkUpdated increments link updated counter.
func (m *InMemoryRecorder) IncLinkUpdated() {
	atomic.AddUint64(&m.linksUpdated, 1)
}

// IncLinkDeleted increments link deleted counter.
func (m *InMemoryRecorder) IncLinkDeleted() {
	atomic.AddUint64(&m.linksDeleted, 1)
}

// IncAnalyticsEventPublished increments event published counter.
func (m *InMemoryRecorder) IncAnalyticsEventPublished(status string) {
	if status == "success" {
		atomic.AddUint64(&m.analyticsEventsPublished, 1)
	} else {
		atomic.AddUint64(&m.analyticsEventsDropped, 1)
	}
}

// IncAnalyticsEventProcessed increments event processed counter.
func (m *InMemoryRecorder) IncAnalyticsEventProcessed(status string) {
	if status == "success" {
		atomic.AddUint64(&m.analyticsEventsProcessed, 1)
		return
	}
	if status == "failed" {
		atomic.AddUint64(&m.analyticsEventsFailed, 1)
		return
	}
	if status == "skipped" {
		atomic.AddUint64(&m.analyticsEventsSkipped, 1)
	}
}

// ObserveAnalyticsBatchSize records batch size.
func (m *InMemoryRecorder) ObserveAnalyticsBatchSize(size int) {
	atomic.AddUint64(&m.analyticsBatchCount, 1)
}

// ObserveAnalyticsBatchDuration records batch processing time.
func (m *InMemoryRecorder) ObserveAnalyticsBatchDuration(duration time.Duration) {
	atomic.AddUint64(&m.analyticsBatchDurationCount, 1)
	atomic.AddInt64(&m.analyticsBatchDurationTotalNs, duration.Nanoseconds())
}

// SetAnalyticsQueueDepth sets the current queue depth.
func (m *InMemoryRecorder) SetAnalyticsQueueDepth(depth int64) {
	atomic.StoreInt64(&m.analyticsQueueDepth, depth)
}

// ObserveAnalyticsIngestLag records ingest lag.
func (m *InMemoryRecorder) ObserveAnalyticsIngestLag(lag time.Duration) {
	atomic.AddUint64(&m.analyticsIngestLagCount, 1)
	atomic.AddInt64(&m.analyticsIngestLagTotalNs, lag.Nanoseconds())
}

// IncWebhookDelivery increments webhook delivery counter by status.
func (m *InMemoryRecorder) IncWebhookDelivery(status string, endpointID string) {
	switch status {
	case "success":
		atomic.AddUint64(&m.webhookDeliveriesSuccess, 1)
	case "failed":
		atomic.AddUint64(&m.webhookDeliveriesFailed, 1)
	case "exhausted":
		atomic.AddUint64(&m.webhookDeliveriesExhausted, 1)
	}
}

// ObserveWebhookDeliveryDuration records webhook delivery duration.
func (m *InMemoryRecorder) ObserveWebhookDeliveryDuration(endpointID string, duration time.Duration) {
	atomic.AddUint64(&m.webhookDurationCount, 1)
	atomic.AddInt64(&m.webhookDurationTotalNs, duration.Nanoseconds())
}

// IncWebhookRetry increments webhook retry counter.
func (m *InMemoryRecorder) IncWebhookRetry(endpointID string, attempt int) {
	atomic.AddUint64(&m.webhookRetries, 1)
}

// SetWebhookQueueDepth sets the webhook queue depth.
func (m *InMemoryRecorder) SetWebhookQueueDepth(depth int64) {
	atomic.StoreInt64(&m.webhookQueueDepth, depth)
}

