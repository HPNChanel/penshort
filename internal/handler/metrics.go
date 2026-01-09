package handler

import (
	"fmt"
	"net/http"

	"github.com/penshort/penshort/internal/metrics"
)

// MetricsHandler exposes in-memory metrics.
type MetricsHandler struct {
	snapshotter metrics.Snapshotter
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(snapshotter metrics.Snapshotter) *MetricsHandler {
	return &MetricsHandler{snapshotter: snapshotter}
}

// Metrics returns metrics in Prometheus exposition format.
func (h *MetricsHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	if h.snapshotter == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	snap := h.snapshotter.Snapshot()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	writeMetric(w, "penshort_redirect_cache_hits_total %d\n", snap.RedirectCacheHits)
	writeMetric(w, "penshort_redirect_cache_misses_total %d\n", snap.RedirectCacheMisses)
	writeMetric(w, "penshort_redirect_duration_seconds_count %d\n", snap.RedirectDurationCount)
	writeMetric(w, "penshort_redirect_duration_seconds_sum %.6f\n", float64(snap.RedirectDurationTotalNs)/1e9)

	writeMetric(w, "penshort_links_created_total %d\n", snap.LinksCreated)
	writeMetric(w, "penshort_links_updated_total %d\n", snap.LinksUpdated)
	writeMetric(w, "penshort_links_deleted_total %d\n", snap.LinksDeleted)

	writeMetric(w, "penshort_analytics_events_published_total{status=\"success\"} %d\n", snap.AnalyticsEventsPublished)
	writeMetric(w, "penshort_analytics_events_published_total{status=\"dropped\"} %d\n", snap.AnalyticsEventsDropped)

	writeMetric(w, "penshort_analytics_events_processed_total{status=\"success\"} %d\n", snap.AnalyticsEventsProcessed)
	writeMetric(w, "penshort_analytics_events_processed_total{status=\"failed\"} %d\n", snap.AnalyticsEventsProcessedFailed)
	writeMetric(w, "penshort_analytics_events_processed_total{status=\"skipped\"} %d\n", snap.AnalyticsEventsProcessedSkipped)

	writeMetric(w, "penshort_analytics_batches_total %d\n", snap.AnalyticsBatchCount)
	writeMetric(w, "penshort_analytics_queue_depth %d\n", snap.AnalyticsQueueDepth)
	writeMetric(w, "penshort_analytics_batch_duration_seconds_count %d\n", snap.AnalyticsBatchDurationCount)
	writeMetric(w, "penshort_analytics_batch_duration_seconds_sum %.6f\n", float64(snap.AnalyticsBatchDurationTotalNs)/1e9)
	writeMetric(w, "penshort_analytics_ingest_lag_seconds_count %d\n", snap.AnalyticsIngestLagCount)
	writeMetric(w, "penshort_analytics_ingest_lag_seconds_sum %.6f\n", float64(snap.AnalyticsIngestLagTotalNs)/1e9)
}

func writeMetric(w http.ResponseWriter, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
