package webhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/penshort/penshort/internal/metrics"
	"github.com/penshort/penshort/internal/model"
)

const (
	// DefaultBatchSize is the number of deliveries to process per poll.
	DefaultBatchSize = 50
	// DefaultPollInterval is the time between polling for pending deliveries.
	DefaultPollInterval = 5 * time.Second
	// DefaultMetricsInterval is how often to update queue depth metrics.
	DefaultMetricsInterval = 10 * time.Second
)

// Worker processes webhook deliveries.
type Worker struct {
	repo            *Repository
	client          *http.Client
	logger          *slog.Logger
	metrics         metrics.Recorder
	batchSize       int
	pollInterval    time.Duration
	metricsInterval time.Duration
	lastMetrics     time.Time
	started         bool
}

// NewWorker creates a new webhook delivery worker.
func NewWorker(repo *Repository, logger *slog.Logger, recorder metrics.Recorder) *Worker {
	if recorder == nil {
		recorder = metrics.NewNoop()
	}
	return &Worker{
		repo:            repo,
		client:          NewHTTPClient(),
		logger:          logger.With("component", "webhook.worker"),
		metrics:         recorder,
		batchSize:       DefaultBatchSize,
		pollInterval:    DefaultPollInterval,
		metricsInterval: DefaultMetricsInterval,
	}
}

// Run starts the worker loop. Blocks until context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	if w.started {
		return errors.New("worker already started")
	}
	w.started = true

	w.logger.Info("webhook worker started")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("webhook worker stopping")
			return ctx.Err()
		case <-ticker.C:
			if err := w.processOnce(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				w.logger.Error("process error", "error", err)
			}
		}
	}
}

// processOnce fetches and processes a batch of pending deliveries.
func (w *Worker) processOnce(ctx context.Context) error {
	w.maybeUpdateQueueDepth(ctx)

	deliveries, err := w.repo.GetPendingDeliveries(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("get pending deliveries: %w", err)
	}

	for _, delivery := range deliveries {
		if err := w.deliver(ctx, delivery); err != nil {
			w.logger.Warn("delivery failed",
				"delivery_id", delivery.ID,
				"error", err,
			)
		}
	}

	return nil
}

// deliver attempts to send a single webhook.
func (w *Worker) deliver(ctx context.Context, delivery *model.WebhookDelivery) error {
	// Get endpoint for target URL and secret
	endpoint, err := w.repo.GetEndpoint(ctx, delivery.EndpointID)
	if err != nil {
		if errors.Is(err, ErrEndpointNotFound) {
			// Endpoint deleted, mark as exhausted
			return w.repo.UpdateDeliveryFailure(ctx, delivery.ID, nil, "endpoint deleted", time.Now(), true)
		}
		return err
	}

	if !endpoint.IsActive() {
		// Endpoint disabled, mark as exhausted
		return w.repo.UpdateDeliveryFailure(ctx, delivery.ID, nil, "endpoint disabled", time.Now(), true)
	}

	// Generate signature with current timestamp
	timestamp := time.Now().Unix()
	signature := GenerateSignature(endpoint.SecretHash, timestamp, []byte(delivery.PayloadJSON))

	// Build request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.TargetURL, bytes.NewReader([]byte(delivery.PayloadJSON)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	SetWebhookHeaders(req, HTTPHeaders{
		Signature:  signature,
		Timestamp:  strconv.FormatInt(timestamp, 10),
		DeliveryID: delivery.ID,
	})

	// Send request
	start := time.Now()
	resp, err := w.client.Do(req)
	duration := time.Since(start)

	w.metrics.ObserveWebhookDeliveryDuration(endpoint.ID, duration)

	if err != nil {
		return w.handleDeliveryError(ctx, delivery, nil, err.Error())
	}
	defer resp.Body.Close()

	// Drain body to allow connection reuse
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.logger.Info("webhook delivered",
			"delivery_id", delivery.ID,
			"target_host", ExtractHost(endpoint.TargetURL),
			"http_status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
		)
		w.metrics.IncWebhookDelivery("success", endpoint.ID)
		return w.repo.UpdateDeliverySuccess(ctx, delivery.ID, resp.StatusCode)
	}

	// Non-2xx response
	return w.handleDeliveryError(ctx, delivery, &resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode))
}

// handleDeliveryError updates delivery status after a failed attempt.
func (w *Worker) handleDeliveryError(ctx context.Context, delivery *model.WebhookDelivery, httpStatus *int, errMsg string) error {
	nextAttempt := delivery.AttemptCount + 1
	exhausted := IsExhausted(nextAttempt, delivery.MaxAttempts)

	status := "failed"
	if exhausted {
		status = "exhausted"
	}

	w.logger.Warn("webhook delivery failed",
		"delivery_id", delivery.ID,
		"attempt", nextAttempt,
		"exhausted", exhausted,
		"error", errMsg,
	)

	w.metrics.IncWebhookDelivery(status, delivery.EndpointID)
	w.metrics.IncWebhookRetry(delivery.EndpointID, nextAttempt)

	nextRetryAt := NextRetryAt(nextAttempt)
	return w.repo.UpdateDeliveryFailure(ctx, delivery.ID, httpStatus, errMsg, nextRetryAt, exhausted)
}

// maybeUpdateQueueDepth periodically updates queue depth metric.
func (w *Worker) maybeUpdateQueueDepth(ctx context.Context) {
	if time.Since(w.lastMetrics) < w.metricsInterval {
		return
	}
	w.lastMetrics = time.Now()

	depth, err := w.repo.GetQueueDepth(ctx)
	if err != nil {
		w.logger.Warn("failed to get queue depth", "error", err)
		return
	}
	w.metrics.SetWebhookQueueDepth(depth)
}

// SetBatchSize overrides the default batch size.
func (w *Worker) SetBatchSize(size int) {
	if size > 0 {
		w.batchSize = size
	}
}

// SetPollInterval overrides the default poll interval.
func (w *Worker) SetPollInterval(interval time.Duration) {
	if interval > 0 {
		w.pollInterval = interval
	}
}
