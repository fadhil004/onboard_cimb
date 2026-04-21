"use strict";

const Redis = require("ioredis");

const redis = new Redis({
  host: process.env.REDIS_HOST || "localhost",
  port: Number(process.env.REDIS_PORT) || 6379,
  password: process.env.REDIS_PASSWORD || undefined,
  lazyConnect: false,
  retryStrategy: (times) => Math.min(times * 100, 3000),
});

redis.on("connect", () => console.log("[Redis] FDS store connected"));
redis.on("error", (err) =>
  console.error("[Redis] FDS store error:", err.message),
);

//  Key helpers

const restrictedKey = (accountNo) => `fds:restricted:${accountNo}`;
const velocityKey = (src, dest) => `fds:velocity:${src}:${dest}`;

// Restriction helpers

async function restrictAccount(accountNo, reason, durationMs = 5 * 60 * 1000) {
  const ttlSec = Math.ceil(durationMs / 1000);

  await redis.set(
    restrictedKey(accountNo),
    JSON.stringify({
      reason: reason || "Temporary restriction",
      restrictedAt: new Date().toISOString(),
    }),
    "EX",
    ttlSec,
  );
}

async function isRestricted(accountNo) {
  const exists = await redis.exists(restrictedKey(accountNo));
  return exists === 1;
}

async function getRestrictionInfo(accountNo) {
  const data = await redis.get(restrictedKey(accountNo));
  if (!data) return null;
  return JSON.parse(data);
}

async function isNewBeneficiary(src, dest) {
  const key = `fds:beneficiaries:${src}`;

  const exists = await redis.sismember(key, dest);

  if (!exists) {
    await redis.sadd(key, dest);
    await redis.expire(key, 24 * 60 * 60); // 1 hari
    return true;
  }

  return false;
}

// Velocity helpers

async function recordAndCountVelocity(sourceAccNo, destAccNo, windowMs) {
  const key = velocityKey(sourceAccNo, destAccNo);
  const now = Date.now();
  const cutoff = now - windowMs;
  const ttlSec = Math.ceil(windowMs / 1000) + 60;

  const pipeline = redis.pipeline();
  pipeline.zremrangebyscore(key, "-inf", cutoff);
  pipeline.zadd(key, now, `${now}-${Math.random()}`);
  pipeline.zcard(key);
  pipeline.expire(key, ttlSec);

  const results = await pipeline.exec();
  return results[2][1];
}

module.exports = {
  restrictAccount,
  isRestricted,
  getRestrictionInfo,
  recordAndCountVelocity,
  isNewBeneficiary,
};
