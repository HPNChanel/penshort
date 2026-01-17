/**
 * Redirect Latency Benchmark
 * 
 * Tests redirect performance with cache hit vs cache miss scenarios.
 * 
 * Usage:
 *   k6 run bench/scripts/redirect-latency.js
 *   k6 run --env BASE_URL=http://localhost:8080 bench/scripts/redirect-latency.js
 * 
 * Prerequisites:
 *   - Docker Compose stack running
 *   - Test links created (run: ./bench/data/setup-links.sh)
 * 
 * @module scripts/redirect-latency
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// =============================================================================
// Configuration
// =============================================================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Pre-generated short codes for cache miss testing
// In real usage, these would come from setup-links.sh output
const CACHE_MISS_CODES = new SharedArray('codes', function () {
    const codes = [];
    for (let i = 0; i < 1000; i++) {
        // Generate random 8-char codes that likely don't exist
        codes.push(randomString(8));
    }
    return codes;
});

// Single code for cache hit testing (created in setup)
let CACHE_HIT_CODE = __ENV.SHORT_CODE || 'bench';

// =============================================================================
// Custom Metrics
// =============================================================================

const cacheHitDuration = new Trend('redirect_cache_hit_duration', true);
const cacheMissDuration = new Trend('redirect_cache_miss_duration', true);
const cacheHitRate = new Rate('redirect_cache_hit_success');
const cacheMissRate = new Rate('redirect_cache_miss_success');
const totalRedirects = new Counter('redirect_total');

// =============================================================================
// k6 Options
// =============================================================================

export const options = {
    scenarios: {
        // Phase 1: Cache Hit Testing (same short code, warm cache)
        cache_hit: {
            executor: 'constant-vus',
            vus: 50,
            duration: '60s',
            exec: 'cacheHitTest',
            startTime: '0s',
            tags: { scenario: 'cache_hit' },
        },
        // Phase 2: Cache Miss Testing (random codes, cold cache)
        cache_miss: {
            executor: 'constant-vus',
            vus: 50,
            duration: '60s',
            exec: 'cacheMissTest',
            startTime: '70s', // Start after cache_hit + 10s cooldown
            tags: { scenario: 'cache_miss' },
        },
    },

    thresholds: {
        // Cache hit thresholds (fast path)
        'http_req_duration{scenario:cache_hit}': ['p95<25', 'p99<50'],
        'redirect_cache_hit_duration': ['p95<25', 'p99<50'],
        'redirect_cache_hit_success': ['rate>0.99'], // >99% success

        // Cache miss thresholds (includes DB lookup)
        'http_req_duration{scenario:cache_miss}': ['p95<150', 'p99<300'],
        'redirect_cache_miss_duration': ['p95<150', 'p99<300'],

        // Overall error rate
        'http_req_failed': ['rate<0.05'], // Allow up to 5% errors (cache miss = 404 expected)
    },
};

// =============================================================================
// Setup: Create test link for cache hit scenario
// =============================================================================

export function setup() {
    console.log(`Base URL: ${BASE_URL}`);

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

    // Try to create a test link for cache hit testing
    const apiKey = __ENV.API_KEY || '';
    if (apiKey) {
        const res = http.post(
            `${BASE_URL}/api/v1/links`,
            JSON.stringify({
                destination: 'https://example.com/benchmark',
                alias: 'bench-perf', // Fixed alias for reproducibility
            }),
            {
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${apiKey}`,
                },
            }
        );

        if (res.status === 201) {
            const body = JSON.parse(res.body);
            CACHE_HIT_CODE = body.short_code;
            console.log(`Created test link: ${CACHE_HIT_CODE}`);
        } else if (res.status === 409) {
            // Alias already exists, use it
            CACHE_HIT_CODE = 'bench-perf';
            console.log(`Using existing test link: ${CACHE_HIT_CODE}`);
        } else {
            console.warn(`Failed to create test link: ${res.status}`);
        }
    }

    // Warm-up: Hit the cache a few times
    console.log('Warming up cache...');
    for (let i = 0; i < 100; i++) {
        http.get(`${BASE_URL}/${CACHE_HIT_CODE}`, {
            redirects: 0, // Don't follow redirects
        });
    }
    console.log('Warm-up complete');

    return { cacheHitCode: CACHE_HIT_CODE };
}

// =============================================================================
// Scenarios
// =============================================================================

/**
 * Cache Hit Test: Repeatedly access the same short code (should be cached)
 */
export function cacheHitTest(data) {
    const url = `${BASE_URL}/${data.cacheHitCode}`;

    const res = http.get(url, {
        redirects: 0, // Don't follow redirects (measure just the redirect response)
        tags: { scenario: 'cache_hit' },
    });

    // Record custom metrics
    cacheHitDuration.add(res.timings.duration);
    totalRedirects.add(1);

    const success = check(res, {
        'status is 301 or 302': (r) => r.status === 301 || r.status === 302,
        'has Location header': (r) => r.headers['Location'] !== undefined,
    });

    cacheHitRate.add(success);

    // Brief pause to avoid overwhelming the server
    sleep(0.01); // 10ms
}

/**
 * Cache Miss Test: Access random short codes (most won't exist = 404)
 * This tests the database lookup path for non-cached codes.
 */
export function cacheMissTest() {
    // Pick a random code from the pool
    const code = CACHE_MISS_CODES[Math.floor(Math.random() * CACHE_MISS_CODES.length)];
    const url = `${BASE_URL}/${code}`;

    const res = http.get(url, {
        redirects: 0,
        tags: { scenario: 'cache_miss' },
    });

    // Record custom metrics (even 404s, as they still measure DB lookup time)
    cacheMissDuration.add(res.timings.duration);
    totalRedirects.add(1);

    // For cache miss, we expect 404 (link not found) - that's OK
    // We're measuring the lookup time, not success
    const success = check(res, {
        'response received': (r) => r.status !== 0, // Any response is success
        'latency reasonable': (r) => r.timings.duration < 500, // <500ms
    });

    cacheMissRate.add(success);

    sleep(0.01); // 10ms
}

// =============================================================================
// Teardown
// =============================================================================

export function teardown(data) {
    console.log('Benchmark complete');
    console.log(`Cache hit test code: ${data.cacheHitCode}`);
    console.log(`Cache miss test codes: ${CACHE_MISS_CODES.length} random codes`);
}

// =============================================================================
// Helpers
// =============================================================================

function randomString(length) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
        result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
}
