/**
 * Worker Ingest Throughput Benchmark
 * 
 * Tests analytics worker performance by generating redirect traffic
 * and monitoring queue processing time.
 * 
 * This benchmark:
 * 1. Creates a burst of redirects (which publish click events)
 * 2. Monitors the Redis stream for queue depth
 * 3. Measures time to drain the queue (worker throughput)
 * 
 * Usage:
 *   k6 run bench/scripts/worker-ingest.js
 *   k6 run --env EVENTS=5000 bench/scripts/worker-ingest.js
 * 
 * Prerequisites:
 *   - Docker Compose stack running
 *   - Analytics worker running
 *   - Test link created
 * 
 * @module scripts/worker-ingest
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Gauge } from 'k6/metrics';

// =============================================================================
// Configuration
// =============================================================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const SHORT_CODE = __ENV.SHORT_CODE || 'bench';
const TOTAL_EVENTS = parseInt(__ENV.EVENTS || '5000', 10);

// =============================================================================
// Custom Metrics
// =============================================================================

const redirectDuration = new Trend('worker_redirect_duration', true);
const eventsGenerated = new Counter('worker_events_generated');
const burstDuration = new Trend('worker_burst_duration', true);

// These would ideally be populated from an external monitoring endpoint
// For now, we measure indirect throughput via redirect response times
const estimatedThroughput = new Gauge('worker_estimated_throughput');

// =============================================================================
// k6 Options
// =============================================================================

export const options = {
    scenarios: {
        // Phase 1: Generate burst of click events
        generate_events: {
            executor: 'shared-iterations',
            vus: 100,
            iterations: TOTAL_EVENTS,
            maxDuration: '120s',
            exec: 'generateEvent',
            tags: { phase: 'generate' },
        },
        // Phase 2: Monitor and wait for processing (runs after burst)
        // In a real scenario, this would query Redis XPENDING
        // For now, we just measure overall timing
        cooldown: {
            executor: 'constant-vus',
            vus: 1,
            duration: '30s',
            startTime: '125s', // After generation burst
            exec: 'monitorQueue',
            tags: { phase: 'monitor' },
        },
    },

    thresholds: {
        // Redirect should still be fast even under load
        'worker_redirect_duration': ['p95<100', 'p99<200'],

        // All events should be generated
        'worker_events_generated': [`count>=${TOTAL_EVENTS * 0.95}`], // 95% of target

        // Overall request handling
        'http_req_failed': ['rate<0.05'],
    },
};

// =============================================================================
// Setup
// =============================================================================

export function setup() {
    console.log(`Base URL: ${BASE_URL}`);
    console.log(`Short code: ${SHORT_CODE}`);
    console.log(`Target events: ${TOTAL_EVENTS}`);

    // Wait for service readiness
    for (let i = 0; i < 30; i++) {
        const res = http.get(`${BASE_URL}/readyz`, { timeout: '5s' });
        if (res.status === 200) {
            console.log('Service is ready');
            break;
        }
        console.log(`Waiting for service... (${i + 1}/30)`);
        sleep(1);
    }

    // Record start time for throughput calculation
    const startTime = Date.now();

    return {
        shortCode: SHORT_CODE,
        startTime: startTime,
        targetEvents: TOTAL_EVENTS,
    };
}

// =============================================================================
// Test Functions
// =============================================================================

/**
 * Generate Event: Make a redirect request (which creates a click event)
 */
export function generateEvent(data) {
    const url = `${BASE_URL}/${data.shortCode}`;

    const res = http.get(url, {
        redirects: 0,
        headers: {
            'User-Agent': `k6-worker-bench/${__VU}-${__ITER}`,
            'Referer': 'https://benchmark.example.com',
        },
    });

    // Record metrics
    redirectDuration.add(res.timings.duration);

    if (res.status === 301 || res.status === 302) {
        eventsGenerated.add(1);
    }

    check(res, {
        'redirect successful': (r) => r.status === 301 || r.status === 302,
    });

    // No sleep - we want maximum burst rate
}

/**
 * Monitor Queue: Check processing status (placeholder)
 * 
 * In a real implementation, this would:
 * 1. Query Redis XPENDING to get queue depth
 * 2. Calculate processing rate
 * 3. Set thresholds based on drain time
 * 
 * For now, we just make health checks and let the timing speak.
 */
export function monitorQueue(data) {
    // Check service health during processing
    const res = http.get(`${BASE_URL}/healthz`);

    check(res, {
        'service healthy during processing': (r) => r.status === 200,
    });

    // Check metrics endpoint if available
    const metricsRes = http.get(`${BASE_URL}/metrics`, { responseType: 'text' });

    if (metricsRes.status === 200) {
        // Parse Prometheus metrics for queue depth (if exposed)
        const body = metricsRes.body;

        // Look for queue depth metric
        const queueMatch = body.match(/penshort_analytics_queue_depth\s+(\d+)/);
        if (queueMatch) {
            const depth = parseInt(queueMatch[1], 10);
            console.log(`Queue depth: ${depth}`);
        }

        // Look for processed events metric
        const processedMatch = body.match(/penshort_analytics_events_processed_total\s+(\d+)/);
        if (processedMatch) {
            const processed = parseInt(processedMatch[1], 10);
            console.log(`Events processed: ${processed}`);
        }
    }

    sleep(2); // Check every 2 seconds
}

// =============================================================================
// Teardown
// =============================================================================

export function teardown(data) {
    const endTime = Date.now();
    const durationSeconds = (endTime - data.startTime) / 1000;

    console.log('Worker ingest benchmark complete');
    console.log(`Total duration: ${durationSeconds.toFixed(2)}s`);
    console.log(`Target events: ${data.targetEvents}`);

    // Estimate throughput (events generated / time taken for generation phase)
    // Note: Actual worker throughput may differ from event generation rate
    const estimatedEPS = data.targetEvents / Math.max(durationSeconds - 30, 1); // Subtract cooldown time
    console.log(`Estimated generation rate: ${estimatedEPS.toFixed(0)} events/sec`);

    console.log('\nTo get actual worker throughput:');
    console.log('  1. Check Redis: XLEN click_events (should be near 0 after processing)');
    console.log('  2. Check Postgres: SELECT COUNT(*) FROM click_events');
    console.log('  3. Compare with target: ' + data.targetEvents);
}
