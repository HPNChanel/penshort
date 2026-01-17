/**
 * Shared k6 utilities for Penshort benchmarks
 * @module util/common
 */

import http from 'k6/http';
import { check, fail } from 'k6';

/**
 * Base URL for API requests (configurable via environment)
 */
export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

/**
 * API key for authenticated requests (configurable via environment)
 */
export const API_KEY = __ENV.API_KEY || '';

/**
 * Common HTTP headers for API requests
 */
export function getApiHeaders() {
  if (!API_KEY) {
    console.warn('API_KEY not set. Some benchmarks may fail.');
  }
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${API_KEY}`,
  };
}

/**
 * Common HTTP headers for public requests (redirects)
 */
export function getPublicHeaders() {
  return {
    'User-Agent': 'k6-benchmark/1.0',
  };
}

/**
 * Wait for service readiness before starting benchmark
 * @returns {boolean} true if service is ready
 */
export function waitForReady(maxRetries = 30, delayMs = 1000) {
  for (let i = 0; i < maxRetries; i++) {
    const res = http.get(`${BASE_URL}/readyz`, { timeout: '5s' });
    if (res.status === 200) {
      console.log(`Service ready after ${i + 1} attempts`);
      return true;
    }
    console.log(`Waiting for service... (${i + 1}/${maxRetries})`);
    // k6 doesn't have sleep in setup, this is handled by scenario delay
  }
  fail('Service not ready after max retries');
  return false;
}

/**
 * Create a test link and return the short code
 * @param {string} destination - Destination URL
 * @returns {string} Short code
 */
export function createTestLink(destination = 'https://example.com/benchmark') {
  const res = http.post(
    `${BASE_URL}/api/v1/links`,
    JSON.stringify({ destination }),
    { headers: getApiHeaders() }
  );

  const success = check(res, {
    'link created': (r) => r.status === 201,
  });

  if (!success) {
    console.error(`Failed to create link: ${res.status} ${res.body}`);
    return null;
  }

  const body = JSON.parse(res.body);
  return body.short_code;
}

/**
 * Load short codes from file (for cache miss testing)
 * @returns {string[]} Array of short codes
 */
export function loadShortCodes() {
  // In k6, we can use SharedArray for large datasets
  // For now, generate random codes inline
  const codes = [];
  for (let i = 0; i < 1000; i++) {
    codes.push(generateRandomCode(8));
  }
  return codes;
}

/**
 * Generate a random alphanumeric code
 * @param {number} length - Code length
 * @returns {string} Random code
 */
export function generateRandomCode(length = 8) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

/**
 * Parse rate limit headers from response
 * @param {object} res - k6 response object
 * @returns {object} Rate limit info
 */
export function parseRateLimitHeaders(res) {
  return {
    limit: parseInt(res.headers['X-Ratelimit-Limit'] || '0', 10),
    remaining: parseInt(res.headers['X-Ratelimit-Remaining'] || '0', 10),
    reset: parseInt(res.headers['X-Ratelimit-Reset'] || '0', 10),
    retryAfter: parseInt(res.headers['Retry-After'] || '0', 10),
  };
}

/**
 * Log rate limit info to console
 * @param {object} rateLimitInfo - Rate limit info object
 */
export function logRateLimitInfo(rateLimitInfo) {
  if (rateLimitInfo.limit > 0) {
    console.log(
      `Rate limit: ${rateLimitInfo.remaining}/${rateLimitInfo.limit}, ` +
      `reset in ${rateLimitInfo.reset - Math.floor(Date.now() / 1000)}s`
    );
  }
}
