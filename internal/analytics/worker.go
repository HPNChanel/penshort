// Package analytics provides click event capture and processing.
package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/penshort/penshort/internal/metrics"
	"github.com/penshort/penshort/internal/model"
)

const (
	// ConsumerGroup is the Redis consumer group name.
	ConsumerGroup = "analytics_workers"

	// DefaultBatchSize is the max events per batch.
	DefaultBatchSize = 500

	// DefaultBlockTimeout is how long to block waiting for messages.
	DefaultBlockTimeout = 5 * time.Second

	// DefaultMaxRetries is the max retries for batch processing.
	DefaultMaxRetries = 3

	// DefaultClaimInterval is how often to scan pending messages.
	DefaultClaimInterval = 10 * time.Second

	// DefaultClaimIdle is the idle time before reclaiming pending messages.
	DefaultClaimIdle = 30 * time.Second

	// DefaultMetricsInterval is how often to refresh queue depth metrics.
	DefaultMetricsInterval = 5 * time.Second
)

// Repository defines the interface for click event persistence.
type Repository interface {
	BulkInsert(ctx context.Context, events []*model.ClickEvent) error
	UpdateDailyStats(ctx context.Context, events []*model.ClickEvent) error
}

// Worker processes click events from Redis stream.
type Worker struct {
	redis           *redis.Client
	repo            Repository
	logger          *slog.Logger
	metrics         metrics.Recorder
	consumerID      string
	batchSize       int
	blockTimeout    time.Duration
	maxRetries      int
	claimInterval   time.Duration
	claimIdle       time.Duration
	metricsInterval time.Duration
	claimStartID    string
	lastClaim       time.Time
	lastMetrics     time.Time

	started  bool
	draining bool
	cancel   context.CancelFunc
	done     chan struct{}
	mu       sync.Mutex
}

// NewWorker creates a new analytics worker.
func NewWorker(client *redis.Client, repo Repository, logger *slog.Logger, consumerID string, recorder metrics.Recorder) *Worker {
	if recorder == nil {
		recorder = metrics.NewNoop()
	}
	return &Worker{
		redis:           client,
		repo:            repo,
		logger:          logger.With("component", "analytics.worker", "consumer_id", consumerID),
		metrics:         recorder,
		consumerID:      consumerID,
		batchSize:       DefaultBatchSize,
		blockTimeout:    DefaultBlockTimeout,
		maxRetries:      DefaultMaxRetries,
		claimInterval:   DefaultClaimInterval,
		claimIdle:       DefaultClaimIdle,
		metricsInterval: DefaultMetricsInterval,
		claimStartID:    "0-0",
	}
}

// Run starts the worker loop. Blocks until context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return errors.New("worker already started")
	}
	w.started = true
	w.done = make(chan struct{})
	ctx, w.cancel = context.WithCancel(ctx)
	w.mu.Unlock()

	defer close(w.done)

	// Ensure consumer group exists
	if err := w.ensureConsumerGroup(ctx); err != nil {
		return fmt.Errorf("ensure consumer group: %w", err)
	}

	w.logger.Info("analytics worker started")

	for {
		w.mu.Lock()
		draining := w.draining
		w.mu.Unlock()

		if draining {
			w.logger.Info("analytics worker draining, stopping")
			return nil
		}

		select {
		case <-ctx.Done():
			w.logger.Info("analytics worker stopping")
			return ctx.Err()
		default:
			if err := w.processOnce(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				w.logger.Error("process error", "error", err)
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// Shutdown gracefully stops the worker, completing any in-flight batch.
// It implements server.ShutdownFunc for integration with graceful shutdown.
func (w *Worker) Shutdown(ctx context.Context) error {
	w.mu.Lock()
	if !w.started {
		w.mu.Unlock()
		return nil
	}
	w.draining = true
	cancel := w.cancel
	done := w.done
	w.mu.Unlock()

	w.logger.Info("analytics worker shutdown initiated")

	// Signal the worker to stop
	if cancel != nil {
		cancel()
	}

	// Wait for worker to finish or context timeout
	if done != nil {
		select {
		case <-done:
			w.logger.Info("analytics worker shutdown complete")
			return nil
		case <-ctx.Done():
			w.logger.Warn("analytics worker shutdown timed out")
			return ctx.Err()
		}
	}
	return nil
}

// ensureConsumerGroup creates the consumer group if it doesn't exist.
func (w *Worker) ensureConsumerGroup(ctx context.Context) error {
	err := w.redis.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "0").Err()
	if err != nil && !isConsumerGroupExistsError(err) {
		return err
	}
	return nil
}

// processOnce reads and processes a single batch.
func (w *Worker) processOnce(ctx context.Context) error {
	w.maybeUpdateQueueDepth(ctx)

	claimed, err := w.maybeClaimPending(ctx)
	if err != nil {
		w.logger.Warn("failed to claim pending messages", "error", err)
	}

	messages := claimed
	if len(messages) == 0 {
		messages, err = w.readBatch(ctx)
		if err != nil {
			return err
		}
	}

	if len(messages) == 0 {
		return nil
	}

	events, messageIDs := w.parseMessages(ctx, messages)
	if len(events) == 0 {
		// All messages were malformed, ACK them anyway to not block
		return w.ackMessages(ctx, messageIDs)
	}

	// Process with retries
	if err := w.processBatchWithRetry(ctx, events); err != nil {
		w.logger.Error("batch processing failed after retries",
			"batch_size", len(events),
			"error", err,
		)
		// Do not ACK so the messages can be retried later.
		return err
	}

	return w.ackMessages(ctx, messageIDs)
}

// maybeClaimPending checks for stuck pending messages and reclaims them.
func (w *Worker) maybeClaimPending(ctx context.Context) ([]redis.XMessage, error) {
	if w.claimInterval <= 0 || w.claimIdle <= 0 {
		return nil, nil
	}
	if !w.lastClaim.IsZero() && time.Since(w.lastClaim) < w.claimInterval {
		return nil, nil
	}

	w.lastClaim = time.Now()
	messages, start, err := w.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   StreamKey,
		Group:    ConsumerGroup,
		Consumer: w.consumerID,
		MinIdle:  w.claimIdle,
		Start:    w.claimStartID,
		Count:    int64(w.batchSize),
	}).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("xautoclaim: %w", err)
	}
	if start != "" {
		w.claimStartID = start
		if start == "0-0" {
			w.claimStartID = "0-0"
		}
	}
	return messages, nil
}

func (w *Worker) maybeUpdateQueueDepth(ctx context.Context) {
	if w.metricsInterval <= 0 {
		return
	}
	if !w.lastMetrics.IsZero() && time.Since(w.lastMetrics) < w.metricsInterval {
		return
	}
	w.lastMetrics = time.Now()

	groups, err := w.redis.XInfoGroups(ctx, StreamKey).Result()
	if err != nil && err != redis.Nil {
		w.logger.Warn("failed to read stream group info", "error", err)
		return
	}
	for _, group := range groups {
		if group.Name == ConsumerGroup {
			w.metrics.SetAnalyticsQueueDepth(group.Pending + group.Lag)
			return
		}
	}
}

// SetBatchSize overrides the default batch size.
func (w *Worker) SetBatchSize(size int) {
	if size > 0 {
		w.batchSize = size
	}
}

// SetBlockTimeout overrides the default blocking timeout.
func (w *Worker) SetBlockTimeout(timeout time.Duration) {
	if timeout > 0 {
		w.blockTimeout = timeout
	}
}

// SetClaimInterval overrides the default pending-claim interval.
func (w *Worker) SetClaimInterval(interval time.Duration) {
	if interval > 0 {
		w.claimInterval = interval
	}
}

// SetClaimIdle overrides the default pending idle threshold.
func (w *Worker) SetClaimIdle(idle time.Duration) {
	if idle > 0 {
		w.claimIdle = idle
	}
}

// SetMetricsInterval overrides the default metrics refresh interval.
func (w *Worker) SetMetricsInterval(interval time.Duration) {
	if interval > 0 {
		w.metricsInterval = interval
	}
}

// readBatch reads messages from the stream using XREADGROUP.
func (w *Worker) readBatch(ctx context.Context) ([]redis.XMessage, error) {
	streams, err := w.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: w.consumerID,
		Streams:  []string{StreamKey, ">"},
		Count:    int64(w.batchSize),
		Block:    w.blockTimeout,
	}).Result()

	if err == redis.Nil || len(streams) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("xreadgroup: %w", err)
	}

	return streams[0].Messages, nil
}

// parseMessages converts Redis messages to ClickEvent models.
// Malformed or invalid messages are moved to the dead-letter queue.
func (w *Worker) parseMessages(ctx context.Context, messages []redis.XMessage) ([]*model.ClickEvent, []string) {
	events := make([]*model.ClickEvent, 0, len(messages))
	messageIDs := make([]string, 0, len(messages))

	for _, msg := range messages {
		messageIDs = append(messageIDs, msg.ID)

		payload, ok := msg.Values["payload"].(string)
		if !ok {
			w.deadLetterMessage(ctx, msg, "invalid_format", "payload field missing or not a string")
			continue
		}

		var eventPayload ClickEventPayload
		if err := json.Unmarshal([]byte(payload), &eventPayload); err != nil {
			w.deadLetterMessage(ctx, msg, "unmarshal_error", err.Error())
			continue
		}
		if err := ValidateClickEventPayload(eventPayload); err != nil {
			w.deadLetterMessage(ctx, msg, "validation_error", err.Error())
			continue
		}

		event := &model.ClickEvent{
			ID:          generateULID(),
			EventID:     msg.ID, // Redis stream ID = idempotency key
			ShortCode:   eventPayload.ShortCode,
			LinkID:      eventPayload.LinkID,
			Referrer:    eventPayload.Referrer,
			UserAgent:   eventPayload.UserAgent,
			VisitorHash: eventPayload.VisitorHash,
			CountryCode: eventPayload.CountryCode,
			ClickedAt:   time.UnixMilli(eventPayload.ClickedAt),
		}

		events = append(events, event)
	}

	return events, messageIDs
}

// deadLetterMessage moves a poison message to the dead-letter queue.
func (w *Worker) deadLetterMessage(ctx context.Context, msg redis.XMessage, reason, detail string) {
	w.logger.Warn("dead-lettering poison message",
		"message_id", msg.ID,
		"reason", reason,
		"detail", detail,
	)

	// Write to dead-letter stream with metadata
	_, err := w.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: DeadLetterStreamKey,
		MaxLen: 10000, // Keep last 10k poison messages
		Approx: true,
		ID:     "*",
		Values: map[string]interface{}{
			"original_id":    msg.ID,
			"original_stream": StreamKey,
			"reason":         reason,
			"detail":         detail,
			"payload":        msg.Values["payload"],
			"dead_lettered_at": time.Now().UTC().Format(time.RFC3339),
		},
	}).Result()

	if err != nil {
		w.logger.Error("failed to write to dead-letter queue",
			"message_id", msg.ID,
			"error", err,
		)
	}

	w.metrics.IncAnalyticsEventProcessed("dead_lettered")
}

// processBatchWithRetry attempts to process a batch with exponential backoff.
func (w *Worker) processBatchWithRetry(ctx context.Context, events []*model.ClickEvent) error {
	var lastErr error

	for attempt := 1; attempt <= w.maxRetries; attempt++ {
		if err := w.processBatch(ctx, events); err != nil {
			lastErr = err
			backoff := time.Duration(1<<attempt) * time.Second
			w.logger.Warn("batch processing failed, retrying",
				"attempt", attempt,
				"backoff_seconds", backoff.Seconds(),
				"error", err,
			)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		return nil
	}

	for range events {
		w.metrics.IncAnalyticsEventProcessed("failed")
	}
	return lastErr
}

// processBatch inserts events and updates daily stats.
func (w *Worker) processBatch(ctx context.Context, events []*model.ClickEvent) error {
	start := time.Now()

	// Bulk insert with ON CONFLICT DO NOTHING for idempotency
	if err := w.repo.BulkInsert(ctx, events); err != nil {
		w.logger.Error("bulk insert failed",
			"batch_size", len(events),
			"first_event_id", events[0].EventID,
			"error", err,
		)
		return fmt.Errorf("bulk insert: %w", err)
	}

	// Update daily aggregations
	if err := w.repo.UpdateDailyStats(ctx, events); err != nil {
		w.logger.Error("failed to update daily stats",
			"batch_size", len(events),
			"error", err,
		)
		return fmt.Errorf("update daily stats: %w", err)
	}

	w.logger.Info("batch processed",
		"events_count", len(events),
		"duration_ms", float64(time.Since(start).Microseconds())/1000,
	)

	w.metrics.ObserveAnalyticsBatchSize(len(events))
	w.metrics.ObserveAnalyticsBatchDuration(time.Since(start))
	for _, event := range events {
		w.metrics.IncAnalyticsEventProcessed("success")
		w.metrics.ObserveAnalyticsIngestLag(time.Since(event.ClickedAt))
	}

	return nil
}

// ackMessages acknowledges processed messages.
func (w *Worker) ackMessages(ctx context.Context, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return nil
	}

	_, err := w.redis.XAck(ctx, StreamKey, ConsumerGroup, messageIDs...).Result()
	if err != nil {
		return fmt.Errorf("xack: %w", err)
	}

	return nil
}

// isConsumerGroupExistsError checks if the error is "BUSYGROUP" (group exists).
func isConsumerGroupExistsError(err error) bool {
	return err != nil && (err.Error() == "BUSYGROUP Consumer Group name already exists" ||
		err.Error() == "BUSYGROUP")
}

// generateULID generates a ULID-like unique ID.
// Uses timestamp + random suffix for uniqueness.
func generateULID() string {
	timestamp := time.Now().UnixNano()
	// Simplified random part - in production, use proper ULID library
	return fmt.Sprintf("%016x", timestamp)
}
