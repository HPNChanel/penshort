/**
 * Centralized threshold definitions for Penshort benchmarks
 * 
 * Philosophy: Thresholds detect REGRESSION (20% tolerance), not absolute targets.
 * These values are baseline-derived and should be updated after collecting
 * 30 days of benchmark data.
 * 
 * @module util/thresholds
 */

/**
 * Baseline performance targets (milliseconds unless noted)
 * 
 * Hardware reference: 4 vCPU, 8GB RAM, Docker Compose on localhost
 */
export const BASELINES = {
    redirect: {
        // Cache hit: Redis lookup only
        cache_hit: {
            p50: 5,    // ms
            p95: 15,   // ms
            p99: 30,   // ms
        },
        // Cache miss: Redis miss + PostgreSQL lookup + cache write
        cache_miss: {
            p50: 25,   // ms
            p95: 75,   // ms
            p99: 150,  // ms
        },
        // Throughput
        rps_cached: 5000,    // requests/sec (cache hit)
        rps_uncached: 1000,  // requests/sec (cache miss)
    },

    api: {
        // Link CRUD operations
        create_link: {
            p95: 150,  // ms
        },
        get_link: {
            p95: 50,   // ms
        },
        list_links: {
            p95: 200,  // ms (includes pagination)
        },
        update_link: {
            p95: 100,  // ms
        },
    },

    rate_limit: {
        // Rate-limited responses should be fast (reject quickly)
        rejection_p95: 10,  // ms
    },

    worker: {
        // Analytics worker throughput
        throughput_min: 1000,  // events/sec
        drain_lag_max: 30,     // seconds (p99)
    },

    general: {
        // Error rate threshold
        error_rate_max: 0.01,  // 1%
    },
};

/**
 * Regression tolerance (20% slower than baseline is acceptable)
 */
export const REGRESSION_TOLERANCE = 0.20;

/**
 * Generate k6 threshold string with regression tolerance
 * @param {number} baseline - Baseline value in ms
 * @param {string} percentile - Percentile (p50, p95, p99)
 * @param {number} tolerance - Tolerance multiplier (default: 0.20)
 * @returns {string} k6 threshold string
 */
export function withTolerance(baseline, percentile = 'p95', tolerance = REGRESSION_TOLERANCE) {
    const threshold = Math.round(baseline * (1 + tolerance));
    return `${percentile}<${threshold}`;
}

/**
 * Generate complete k6 thresholds object for redirect benchmark
 * @returns {object} k6 thresholds configuration
 */
export function getRedirectThresholds() {
    return {
        // Cache hit scenario
        'http_req_duration{scenario:cache_hit}': [
            withTolerance(BASELINES.redirect.cache_hit.p95, 'p95'),
            withTolerance(BASELINES.redirect.cache_hit.p99, 'p99'),
        ],
        // Cache miss scenario
        'http_req_duration{scenario:cache_miss}': [
            withTolerance(BASELINES.redirect.cache_miss.p95, 'p95'),
            withTolerance(BASELINES.redirect.cache_miss.p99, 'p99'),
        ],
        // Error rate
        'http_req_failed': [`rate<${BASELINES.general.error_rate_max}`],
    };
}

/**
 * Generate complete k6 thresholds object for API benchmark
 * @returns {object} k6 thresholds configuration
 */
export function getApiThresholds() {
    return {
        'http_req_duration{endpoint:create_link}': [
            withTolerance(BASELINES.api.create_link.p95, 'p95'),
        ],
        'http_req_duration{endpoint:get_link}': [
            withTolerance(BASELINES.api.get_link.p95, 'p95'),
        ],
        'http_req_duration{endpoint:list_links}': [
            withTolerance(BASELINES.api.list_links.p95, 'p95'),
        ],
        'http_req_failed': [`rate<${BASELINES.general.error_rate_max}`],
    };
}

/**
 * Generate complete k6 thresholds object for rate limit benchmark
 * @returns {object} k6 thresholds configuration
 */
export function getRateLimitThresholds() {
    return {
        // Rate-limited responses should be fast
        'http_req_duration{status:429}': [
            withTolerance(BASELINES.rate_limit.rejection_p95, 'p95'),
        ],
        // Ensure rate limiting actually triggers
        'rate_limit_triggered': ['count>50'],
    };
}

/**
 * Generate complete k6 thresholds object for worker benchmark
 * @returns {object} k6 thresholds configuration
 */
export function getWorkerThresholds() {
    return {
        // Worker should drain queue within time limit
        'worker_drain_time': [`value<${BASELINES.worker.drain_lag_max}`],
        // Minimum throughput
        'worker_throughput': [`value>${BASELINES.worker.throughput_min}`],
    };
}
