'use strict';

require('dotenv').config();

const grpc = require('@grpc/grpc-js');
const { createGrpcServer } = require('./grpc/server');
const logger = require('./rules/logger');

const GRPC_PORT = process.env.GRPC_PORT || '50052';

const server = createGrpcServer();

server.bindAsync(
  `0.0.0.0:${GRPC_PORT}`,
  grpc.ServerCredentials.createInsecure(),
  (err, port) => {
    if (err) {
      logger.error('Failed to bind gRPC server', { err: err.message });
      process.exit(1);
    }
    logger.info(`[gRPC] Fraud Detection Service listening on port ${port}`);
  }
);

process.on('SIGTERM', () => {
  logger.info('SIGTERM received — shutting down');
  server.tryShutdown(() => {
    logger.info('gRPC server shut down');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  logger.info('SIGINT received — shutting down');
  server.tryShutdown(() => process.exit(0));
});
