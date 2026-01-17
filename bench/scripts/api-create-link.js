/**
 * API Create Link Throughput Benchmark
 * 
 * Tests link creation performance under different API key rate limit tiers.
 * Simulates multiple API keys creating links at their allowed rates.
 * 
 * Usage:
 *   k6 run bench/scripts/api-create-link.js
 *   k6 run --env API_KEY=xxx bench/scripts/api-create-link.js
 * 
 * Prerequisites:
 *   - Docker Compose stack running
 *   - API key(s) bootstrapped
 * 
 * @module scripts/api-create-link
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';

// =============================================================================
// Configuration
// =============================================================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || '';

// =============================================================================
// Custom Metrics
// =============================================================================

const createDuration = new Trend('api_create_link_duration', true);
const getDuration = new Trend('api_get_link_duration', true);
const listDuration = new Trend('api_list_links_duration', true);
const apiRateLimited = new Counter('api_rate_limited');
const apiSuccess = new Rate('api_success_rate');

// =============================================================================
// k6 Options
// =============================================================================

export const options = {
    scenarios: {
        // Sustained load: Create links at moderate rate
        sustained_create: {
            executor: 'constant-arrival-rate',
            rate: 30, // 30 requests per second (below typical rate limit)
            timeUnit: '1s',
            duration: '60s',
            preAllocatedVUs: 50,
            maxVUs: 100,
            exec: 'createLinkTest',
            tags: { endpoint: 'create_link' },
        },
        // Burst load: Create many links quickly
        burst_create: {
            executor: 'ramping-arrival-rate',
            startRate: 10,
            timeUnit: '1s',
            preAllocatedVUs: 100,
            maxVUs: 200,
            stages: [
                { duration: '10s', target: 50 },
                { duration: '20s', target: 100 },  // Push towards rate limit
                { duration: '10s', target: 30 },
            ],
            startTime: '70s', // After sustained test
            exec: 'createLinkTest',
            tags: { endpoint: 'create_link_burst' },
        },
        // Read test: Get and list links
        read_links: {
            executor: 'constant-vus',
            vus: 20,
            duration: '30s',
            startTime: '140s', // After create tests
            exec: 'readLinksTest',
            tags: { endpoint: 'read_links' },
        },
    },

    thresholds: {
        // Create link latency
        'api_create_link_duration': ['p95<200', 'p99<500'],
        'http_req_duration{endpoint:create_link}': ['p95<200'],
        'http_req_duration{endpoint:create_link_burst}': ['p95<300'],

        // Read link latency
        'api_get_link_duration': ['p95<100'],
        'api_list_links_duration': ['p95<300'],

        // Success rate (excluding rate limit responses)
        'api_success_rate': ['rate>0.95'], // >95% success

        // Overall error rate
        'http_req_failed': ['rate<0.05'],
    },
};

// =============================================================================
// Setup
// =============================================================================

export function setup() {
    console.log(`Base URL: ${BASE_URL}`);

    if (!API_KEY) {
        console.warn('No API_KEY set. Run: export API_KEY=$(go run ./scripts/bootstrap-api-key.go)');
    }

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

    // Store created link IDs for read tests
    const createdLinks = [];

    // Create a few initial links for read testing
    for (let i = 0; i < 10; i++) {
        const res = http.post(
            `${BASE_URL}/api/v1/links`,
            JSON.stringify({
                destination: `https://example.com/setup-${i}`,
            }),
            { headers: getHeaders() }
        );

        if (res.status === 201) {
            const body = JSON.parse(res.body);
            createdLinks.push(body.id);
        }
    }

    console.log(`Setup: Created ${createdLinks.length} links for read testing`);
    return { createdLinks };
}

// =============================================================================
// Test Functions
// =============================================================================

/**
 * Create Link Test: POST /api/v1/links
 */
export function createLinkTest() {
    const destination = `https://example.com/bench-${Date.now()}-${randomInt(10000)}`;

    const res = http.post(
        `${BASE_URL}/api/v1/links`,
        JSON.stringify({ destination }),
        { headers: getHeaders() }
    );

    // Record metrics
    createDuration.add(res.timings.duration);

    if (res.status === 429) {
        apiRateLimited.add(1);
        apiSuccess.add(0);

        check(res, {
            'rate limit response fast': (r) => r.timings.duration < 50,
        });
    } else if (res.status === 201) {
        apiSuccess.add(1);

        check(res, {
            'link created': (r) => r.status === 201,
            'has id': (r) => JSON.parse(r.body).id !== undefined,
            'has short_code': (r) => JSON.parse(r.body).short_code !== undefined,
        });
    } else {
        apiSuccess.add(0);
        console.warn(`Unexpected status: ${res.status}`);
    }
}

/**
 * Read Links Test: GET /api/v1/links and GET /api/v1/links/{id}
 */
export function readLinksTest(data) {
    // Test 1: List links
    const listRes = http.get(`${BASE_URL}/api/v1/links?limit=20`, {
        headers: getHeaders(),
        tags: { endpoint: 'list_links' },
    });

    listDuration.add(listRes.timings.duration);

    check(listRes, {
        'list returns 200': (r) => r.status === 200,
        'list has links': (r) => JSON.parse(r.body).links !== undefined,
    });

    // Test 2: Get specific link (if we have created links)
    if (data.createdLinks && data.createdLinks.length > 0) {
        const linkId = data.createdLinks[randomInt(data.createdLinks.length)];

        const getRes = http.get(`${BASE_URL}/api/v1/links/${linkId}`, {
            headers: getHeaders(),
            tags: { endpoint: 'get_link' },
        });

        getDuration.add(getRes.timings.duration);

        check(getRes, {
            'get returns 200': (r) => r.status === 200,
            'get has id': (r) => JSON.parse(r.body).id !== undefined,
        });
    }

    sleep(0.1); // 100ms between read operations
}

// =============================================================================
// Teardown
// =============================================================================

export function teardown(data) {
    console.log('API benchmark complete');
    console.log(`Created ${data.createdLinks?.length || 0} setup links`);
}

// =============================================================================
// Helpers
// =============================================================================

function getHeaders() {
    return {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${API_KEY}`,
    };
}

function randomInt(max) {
    return Math.floor(Math.random() * max);
}
