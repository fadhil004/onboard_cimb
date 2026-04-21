"use strict";

const store = require("../store");
const logger = require("./logger");

const RULES = {
  MAX_AMOUNT: Number(process.env.RULE_MAX_AMOUNT) || 1000000,
  VELOCITY_THRESHOLD: Number(process.env.RULE_VELOCITY_THRESHOLD) || 5,
  VELOCITY_WINDOW_MS:
    Number(process.env.RULE_VELOCITY_WINDOW_MS) || 5 * 60 * 1000,
  VELOCITY_AUTO_BLOCK: process.env.RULE_VELOCITY_AUTO_BLOCK !== "false",
};

/**
 * Run all fraud rules against a candidate transaction.
 * Returns { allowed, fraudCode, message, riskLevel }
 */
function runFraudChecks({ sourceAccountNo, beneficiaryAccountNo, amount }) {
  // Rule 1 — Source blocked
  if (store.isBlocked(sourceAccountNo)) {
    const info = store.getBlockInfo(sourceAccountNo);
    logger.warn("BLOCKED_ACCOUNT hit", { sourceAccountNo });
    return {
      allowed: false,
      fraudCode: "BLOCKED_ACCOUNT",
      message: `Source account ${sourceAccountNo} is blocked: ${info.reason}`,
      riskLevel: "CRITICAL",
    };
  }

  // Rule 2 — Beneficiary blocked
  if (store.isBlocked(beneficiaryAccountNo)) {
    const info = store.getBlockInfo(beneficiaryAccountNo);
    logger.warn("BLOCKED_BENEFICIARY hit", { beneficiaryAccountNo });
    return {
      allowed: false,
      fraudCode: "BLOCKED_BENEFICIARY",
      message: `Beneficiary account ${beneficiaryAccountNo} is blocked: ${info.reason}`,
      riskLevel: "HIGH",
    };
  }

  // Rule 3 — Amount ceiling
  if (amount > RULES.MAX_AMOUNT) {
    logger.warn("AMOUNT_EXCEEDED", { amount, limit: RULES.MAX_AMOUNT });
    return {
      allowed: false,
      fraudCode: "AMOUNT_EXCEEDED",
      message: `Transfer amount ${amount} exceeds the maximum allowed ${RULES.MAX_AMOUNT}`,
      riskLevel: "HIGH",
    };
  }

  // Rule 4 — Velocity / Actimize
  const velocityCount = store.recordAndCountVelocity(
    sourceAccountNo,
    beneficiaryAccountNo,
    RULES.VELOCITY_WINDOW_MS,
  );

  if (velocityCount >= RULES.VELOCITY_THRESHOLD) {
    logger.warn("VELOCITY_BREACH detected", {
      sourceAccountNo,
      beneficiaryAccountNo,
      velocityCount,
      threshold: RULES.VELOCITY_THRESHOLD,
      windowMs: RULES.VELOCITY_WINDOW_MS,
    });

    if (RULES.VELOCITY_AUTO_BLOCK && !store.isBlocked(sourceAccountNo)) {
      store.blockAccount(
        sourceAccountNo,
        `Auto-actimized: ${velocityCount} transfers to ${beneficiaryAccountNo} within ${RULES.VELOCITY_WINDOW_MS / 1000}s`,
        "SYSTEM:ACTIMIZE",
      );
      logger.warn("Account auto-blocked by actimize rule", { sourceAccountNo });
    }

    return {
      allowed: false,
      fraudCode: "VELOCITY_BREACH",
      message: `Suspicious activity: ${velocityCount} transfers to ${beneficiaryAccountNo} within ${RULES.VELOCITY_WINDOW_MS / 1000}s`,
      riskLevel: "CRITICAL",
    };
  }

  // Risk scoring (allowed but elevated)
  let riskLevel = "LOW";
  if (amount > RULES.MAX_AMOUNT * 0.7) riskLevel = "MEDIUM";
  if (velocityCount >= Math.floor(RULES.VELOCITY_THRESHOLD * 0.6))
    riskLevel = "MEDIUM";

  return {
    allowed: true,
    fraudCode: "OK",
    message: "Transaction cleared",
    riskLevel,
  };
}

module.exports = { runFraudChecks, RULES };
