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
}

// NewInMemory returns a Recorder that stores counters in memory.
func NewInMemory() *InMemoryRecorder {
	return &InMemoryRecorder{}
}

// Snapshot returns a copy of the counters.
func (m *InMemoryRecorder) Snapshot() Snapshot {
	return Snapshot{
		RedirectCacheHits:       atomic.LoadUint64(&m.redirectCacheHits),
		RedirectCacheMisses:     atomic.LoadUint64(&m.redirectCacheMisses),
		RedirectDurationCount:   atomic.LoadUint64(&m.redirectDurationCount),
		RedirectDurationTotalNs: atomic.LoadInt64(&m.redirectDurationTotalNs),
		LinksCreated:            atomic.LoadUint64(&m.linksCreated),
		LinksUpdated:            atomic.LoadUint64(&m.linksUpdated),
		LinksDeleted:            atomic.LoadUint64(&m.linksDeleted),
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
