"use strict";

const store = require("../store");
const logger = require("./logger");

const RULES = {
  MAX_AMOUNT: Number(process.env.RULE_MAX_AMOUNT) || 1_000_000,
  VELOCITY_THRESHOLD: Number(process.env.RULE_VELOCITY_THRESHOLD) || 5,
  VELOCITY_WINDOW_MS:
    Number(process.env.RULE_VELOCITY_WINDOW_MS) || 5 * 60 * 1000,
  RESTRICT_DURATION_MS:
    Number(process.env.RULE_RESTRICT_DURATION_MS) || 5 * 60 * 1000,
};

// Helper
function getRiskLevel(score) {
  if (score >= 80) return "CRITICAL";
  if (score >= 60) return "HIGH";
  if (score >= 30) return "MEDIUM";
  return "LOW";
}

function getDecision(score) {
  if (score >= 80) return "REJECT";
  if (score >= 60) return "REVIEW";
  return "ALLOW";
}

async function runFraudChecks({
  sourceAccountNo,
  beneficiaryAccountNo,
  amount,
}) {
  let score = 0;
  let fraudCode = "OK";
  const reasons = [];

  // 1. Restricted check
  if (await store.isRestricted(sourceAccountNo)) {
    return {
      allowed: false,
      fraudCode: "ACCOUNT_RESTRICTED",
      message: "Account temporarily restricted",
      riskLevel: "CRITICAL",
      score: 100,
      decision: "REJECT",
    };
  }

  // 2. Velocity
  const velocityCount = await store.recordAndCountVelocity(
    sourceAccountNo,
    beneficiaryAccountNo,
    RULES.VELOCITY_WINDOW_MS,
  );

  if (velocityCount >= RULES.VELOCITY_THRESHOLD) {
    const alreadyRestricted = await store.isRestricted(sourceAccountNo);

    if (!alreadyRestricted) {
      await store.restrictAccount(
        sourceAccountNo,
        `Fraud detected: ${reasons.join(", ")}`,
        RULES.RESTRICT_DURATION_MS,
      );
    }

    logger.warn("Velocity anomaly", {
      sourceAccountNo,
      score,
      reasons,
    });
  }

  // 3. Amount anomaly
  if (amount > RULES.MAX_AMOUNT) {
    return {
      allowed: false,
      fraudCode: "MAX_AMOUNT_EXCEEDED",
      message: "Max amount exceeded",
      riskLevel: "HIGH",
      score: 100,
      decision: "REVIEW",
    };
  } else if (amount > RULES.MAX_AMOUNT * 0.8) {
    score += 25;
    reasons.push("High amount");
  }

  // 4. Round number anomaly
  if (amount % 100000 === 0) {
    score += 5;
    reasons.push("Round number suspicious");
  }

  // 5. Time anomaly
  const hour = new Date().getHours();
  if (hour >= 1 && hour <= 4) {
    score += 5;
    reasons.push("Odd transaction hour");
  }

  // 6. New beneficiary
  const isNew = await store.isNewBeneficiary(
    sourceAccountNo,
    beneficiaryAccountNo,
  );

  if (isNew) {
    score += 5;
    reasons.push("New beneficiary");
  }

  // Final decision
  score = Math.min(score, 100);
  const riskLevel = getRiskLevel(score);
  const decision = getDecision(score);

  if (riskLevel === "CRITICAl") {
    const alreadyRestricted = await store.isRestricted(sourceAccountNo);

    if (!alreadyRestricted) {
      await store.restrictAccount(
        sourceAccountNo,
        `Fraud detected: ${reasons.join(", ")}`,
        RULES.RESTRICT_DURATION_MS,
      );
    }

    logger.warn("ACCOUNT_RESTRICTED_BY_SCORE", {
      sourceAccountNo,
      score,
      reasons,
    });
  }

  return {
    allowed: decision === "ALLOW",
    fraudCode,
    message: reasons.join(", ") || "Transaction normal",
    riskLevel,
    score,
    decision,
  };
}

module.exports = { runFraudChecks };
