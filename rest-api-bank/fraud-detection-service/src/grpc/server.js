"use strict";

const path = require("path");
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const { runFraudChecks } = require("../rules/fraudRules");
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
async function checkTransaction(call, callback) {
  const req = call.request;

  logger.info("CheckTransaction", {
    source: req.source_account_no,
    beneficiary: req.beneficiary_account_no,
    amount: req.amount,
  });

  try {
    const result = await runFraudChecks({
      sourceAccountNo: req.source_account_no,
      beneficiaryAccountNo: req.beneficiary_account_no,
      amount: req.amount,
    });

    logger.info("FraudCheck result", result);

    callback(null, {
      allowed: result.allowed,
      fraud_code: result.fraudCode,
      message: result.message,
      risk_level: result.riskLevel,
      score: result.score || 0,
      decision: result.decision || "",
    });
  } catch (err) {
    logger.error("CheckTransaction error", { err: err.message });
    callback({
      code: grpc.status.INTERNAL,
      message: "Internal fraud check error",
    });
  }
}

// Server factory
function createGrpcServer() {
  const server = new grpc.Server();
  server.addService(fraudProto.FraudDetectionService.service, {
    CheckTransaction: checkTransaction,
  });
  return server;
}

module.exports = { createGrpcServer };
