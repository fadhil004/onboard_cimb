'use strict';

/**
 * store.js — in-memory state for blocked accounts and velocity tracking.
 * In production: replace with Redis or a DB.
 */

// Map<accountNo, { reason, blockedBy, blockedAt }>
const blockedAccounts = new Map();

// Map<"src:dest", { timestamps: number[] }>
const velocityMap = new Map();

// ── Blocked account helpers ───────────────────────────────────────────────────

function blockAccount(accountNo, reason, blockedBy) {
  blockedAccounts.set(accountNo, {
    reason,
    blockedBy,
    blockedAt: new Date().toISOString(),
  });
}

function unblockAccount(accountNo) {
  return blockedAccounts.delete(accountNo);
}

function isBlocked(accountNo) {
  return blockedAccounts.has(accountNo);
}

function getBlockInfo(accountNo) {
  return blockedAccounts.get(accountNo) || null;
}

function listBlocked() {
  const result = [];
  for (const [accountNo, info] of blockedAccounts.entries()) {
    result.push({ account_no: accountNo, ...info });
  }
  return result;
}

// ── Velocity helpers ──────────────────────────────────────────────────────────

/**
 * Record a transfer attempt and return the count within the sliding window.
 */
function recordAndCountVelocity(sourceAccNo, destAccNo, windowMs) {
  const key = `${sourceAccNo}:${destAccNo}`;
  const now = Date.now();
  const cutoff = now - windowMs;

  if (!velocityMap.has(key)) {
    velocityMap.set(key, { timestamps: [] });
  }

  const entry = velocityMap.get(key);
  entry.timestamps = entry.timestamps.filter((t) => t > cutoff);
  entry.timestamps.push(now);
  return entry.timestamps.length;
}

module.exports = {
  blockAccount,
  unblockAccount,
  isBlocked,
  getBlockInfo,
  listBlocked,
  recordAndCountVelocity,
};
