/**
 * helpers/utils.js
 * Shared utilities: SNAP headers, UUID v4, common checks.
 */

import { check } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// ─── SNAP BI header builder ─────────────────────────────────────────────────
export function snapHeaders(externalID) {
  return {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer test-token-k6',
    'X-SIGNATURE': 'test-signature-k6',
    'X-PARTNER-ID': 'PARTNER-K6-TEST',
    'X-EXTERNAL-ID': externalID || uuidv4(),
    'CHANNEL-ID': 'K6',
    'X-TIMESTAMP': new Date().toISOString(),
  };
}

// ─── Plain JSON headers (no SNAP required) ──────────────────────────────────
export function jsonHeaders() {
  return { 'Content-Type': 'application/json' };
}

// ─── Random integer between min..max (inclusive) ────────────────────────────
export function randInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// ─── Random element from array ──────────────────────────────────────────────
export function pick(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

// ─── Standard response checks ───────────────────────────────────────────────
export function checkOK(res, label) {
  return check(res, {
    [`${label} - status 200`]: (r) => r.status === 200,
    [`${label} - body not empty`]: (r) => r.body && r.body.length > 0,
  });
}

export function checkCreated(res, label) {
  return check(res, {
    [`${label} - status 200`]: (r) => r.status === 200,
    [`${label} - has responseCode`]: (r) => {
      try { return JSON.parse(r.body).responseCode !== undefined; } catch { return false; }
    },
  });
}

export function checkTransfer(res, label) {
  return check(res, {
    [`${label} - status 200`]: (r) => r.status === 200,
    [`${label} - referenceNo present`]: (r) => {
      try { return JSON.parse(r.body).referenceNo !== undefined; } catch { return false; }
    },
    [`${label} - responseCode 2xx`]: (r) => {
      try { return JSON.parse(r.body).responseCode?.startsWith('2'); } catch { return false; }
    },
  });
}

// ─── ISO-8601 timestamp for transactionDate ─────────────────────────────────
export function nowISO() {
  return new Date().toISOString().replace('T', ' ').substring(0, 19);
}

// ─── Re-export uuidv4 for convenience ───────────────────────────────────────
export { uuidv4 };
