"use strict";

const path = require("path");
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const { runFraudChecks } = require("../rules/fraudRules");
const store = require("../store");
const logger = require("../rules/logger");

const PROTO_PATH = path.resolve(__dirname, "../../proto/fraud.proto");

const packageDef = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const fraudProto = grpc.loadPackageDefinition(packageDef).fraud;

// RPC: CheckTransaction

function checkTransaction(call, callback) {
  const req = call.request;

  logger.info("CheckTransaction", {
    source: req.source_account_no,
    beneficiary: req.beneficiary_account_no,
    amount: req.amount,
    currency: req.currency,
    ref: req.partner_reference_no,
  });

  try {
    const result = runFraudChecks({
      sourceAccountNo: req.source_account_no,
      beneficiaryAccountNo: req.beneficiary_account_no,
      amount: req.amount,
    });

    logger.info("FraudCheck result", {
      allowed: result.allowed,
      fraudCode: result.fraudCode,
      riskLevel: result.riskLevel,
    });

    callback(null, {
      allowed: result.allowed,
      fraud_code: result.fraudCode,
      message: result.message,
      risk_level: result.riskLevel,
    });
  } catch (err) {
    logger.error("CheckTransaction error", { err: err.message });
    callback({
      code: grpc.status.INTERNAL,
      message: "Internal fraud check error",
    });
  }
}

// RPC: BlockAccount

function blockAccount(call, callback) {
  const { account_no, reason, blocked_by } = call.request;
  logger.info("BlockAccount", { account_no, reason, blocked_by });

  if (!account_no) {
    return callback({
      code: grpc.status.INVALID_ARGUMENT,
      message: "account_no is required",
    });
  }

  store.blockAccount(
    account_no,
    reason || "Manual block",
    blocked_by || "OPERATOR",
  );
  callback(null, {
    success: true,
    message: `Account ${account_no} has been blocked`,
  });
}

// ── RPC: UnblockAccount ───────────────────────────────────────────────────────

function unblockAccount(call, callback) {
  const { account_no, unblocked_by } = call.request;
  logger.info("UnblockAccount", { account_no, unblocked_by });

  if (!account_no) {
    return callback({
      code: grpc.status.INVALID_ARGUMENT,
      message: "account_no is required",
    });
  }

  const existed = store.unblockAccount(account_no);
  callback(null, {
    success: existed,
    message: existed
      ? `Account ${account_no} unblocked by ${unblocked_by || "OPERATOR"}`
      : `Account ${account_no} was not blocked`,
  });
}

// ── RPC: GetAccountStatus ─────────────────────────────────────────────────────

function getAccountStatus(call, callback) {
  const { account_no } = call.request;
  const blocked = store.isBlocked(account_no);
  const info = store.getBlockInfo(account_no);

  callback(null, {
    account_no,
    is_blocked: blocked,
    block_reason: info ? info.reason : "",
    blocked_by: info ? info.blockedBy : "",
    blocked_at: info ? info.blockedAt : "",
    velocity_count: 0,
  });
}

// ── Server factory ────────────────────────────────────────────────────────────

function createGrpcServer() {
  const server = new grpc.Server();
  server.addService(fraudProto.FraudDetectionService.service, {
    CheckTransaction: checkTransaction,
    BlockAccount: blockAccount,
    UnblockAccount: unblockAccount,
    GetAccountStatus: getAccountStatus,
  });
  return server;
}

module.exports = { createGrpcServer };
