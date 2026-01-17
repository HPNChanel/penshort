/**
 * Redirect Rate Limiting Benchmark
 * 
 * Tests IP-based rate limiting on the redirect endpoint.
 * Ramps up request rate to trigger 429 responses and measures rejection latency.
 * 
 * Usage:
 *   k6 run bench/scripts/redirect-ratelimit.js
 *   k6 run --env REDIRECT_RPS=100 bench/scripts/redirect-ratelimit.js
 * 
 * Prerequisites:
 *   - Docker Compose stack running
 *   - Rate limiting enabled (REDIRECT_RATE_LIMIT_ENABLED=true)
 * 
 * @module scripts/redirect-ratelimit
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';

// =============================================================================
// Configuration
// =============================================================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const SHORT_CODE = __ENV.SHORT_CODE || 'bench';

// Expected rate limit (default: 100 RPS per IP)
const EXPECTED_LIMIT = parseInt(__ENV.REDIRECT_RPS || '100', 10);

// =============================================================================
// Custom Metrics
// =============================================================================

const rateLimitHits = new Counter('rate_limit_triggered');
const rejectionLatency = new Trend('rate_limit_rejection_duration', true);
const successLatency = new Trend('rate_limit_success_duration', true);
const hitRate = new Rate('rate_limit_hit_rate');

// =============================================================================
// k6 Options
// =============================================================================

export const options = {
    scenarios: {
        // Ramp up to exceed rate limit
        ramp_up: {
            executor: 'ramping-arrival-rate',
            startRate: 10,
            timeUnit: '1s',
            preAllocatedVUs: 200,
            maxVUs: 500,
            stages: [
                { duration: '10s', target: 50 },    // Warm up
                { duration: '20s', target: 150 },   // Exceed limit (assuming 100 RPS)
                { duration: '20s', target: 200 },   // Push harder
                { duration: '10s', target: 50 },    // Cool down
            ],
        },
    },

    thresholds: {
        // Rate-limited responses should be very fast (reject immediately)
        'rate_limit_rejection_duration': ['p95<20', 'p99<50'],

        // Ensure rate limiting actually triggers
        'rate_limit_triggered': ['count>50'],

        // Success path shouldn't be too slow
        'rate_limit_success_duration': ['p95<100'],

        // Overall request handling
        'http_req_failed': ['rate<0.01'], // Only count actual failures, not 429s
    },
};

// =============================================================================
// Setup
// =============================================================================

export function setup() {
    console.log(`Base URL: ${BASE_URL}`);
    console.log(`Short code: ${SHORT_CODE}`);
    console.log(`Expected rate limit: ${EXPECTED_LIMIT} RPS`);

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

    return { shortCode: SHORT_CODE };
}

// =============================================================================
// Main Test
// =============================================================================

export default function (data) {
    const url = `${BASE_URL}/${data.shortCode}`;

    const res = http.get(url, {
        redirects: 0,
        tags: { name: 'redirect' },
    });

    // Check for rate limiting
    if (res.status === 429) {
        // Rate limited - record metrics
        rateLimitHits.add(1);
        rejectionLatency.add(res.timings.duration);
        hitRate.add(1);

        check(res, {
            'rate limit response is fast': (r) => r.timings.duration < 50,
            'has Retry-After header': (r) => r.headers['Retry-After'] !== undefined,
        });

        // Parse Retry-After for debugging
        const retryAfter = res.headers['Retry-After'];
        if (__ENV.DEBUG) {
            console.log(`Rate limited. Retry after: ${retryAfter}s`);
        }
    } else {
        // Success - record metrics
        successLatency.add(res.timings.duration);
        hitRate.add(0);

        check(res, {
            'redirect succeeded': (r) => r.status === 301 || r.status === 302,
        });
    }
}

// =============================================================================
// Teardown
// =============================================================================

export function teardown(data) {
    console.log('Rate limit benchmark complete');
    console.log(`Tested against: ${BASE_URL}/${data.shortCode}`);
}
