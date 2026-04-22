/**
 * helpers/thresholds.js
 * Reusable SLO thresholds — import into each scenario.
 */

// Account-service thresholds (simpler CRUD, should be faster)
export const accountThresholds = {
  // p95 latency under 500ms
  'http_req_duration{service:account}': ['p(95)<500'],
  // p99 latency under 1s
  'http_req_duration{service:account}': ['p(99)<1000'],
  // Error rate under 2%
  'http_req_failed{service:account}': ['rate<0.02'],
};

// Transaction-service thresholds (involves gRPC to account + fraud, more complex)
export const transactionThresholds = {
  // p95 latency under 1.5s (crosses 2 gRPC services)
  'http_req_duration{service:transaction}': ['p(95)<1500'],
  // p99 latency under 3s
  'http_req_duration{service:transaction}': ['p(99)<3000'],
  // Error rate under 3%
  'http_req_failed{service:transaction}': ['rate<0.03'],
};

// Combined thresholds for the full suite
export const allThresholds = {
  // Overall p95 < 1s
  http_req_duration: ['p(95)<1500', 'p(99)<3000'],
  // Overall error rate < 3%
  http_req_failed: ['rate<0.03'],
  // Checks must pass > 97%
  checks: ['rate>0.97'],
};
