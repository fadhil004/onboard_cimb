'use strict';

const { NodeSDK } = require('@opentelemetry/sdk-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');
const { GrpcInstrumentation } = require('@opentelemetry/instrumentation-grpc');
const { Resource } = require('@opentelemetry/resources');
const { ATTR_SERVICE_NAME } = require('@opentelemetry/semantic-conventions');

const serviceName = process.env.SERVICE_NAME || 'fraud-detection-service';
const otlpEndpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'http://tempo:4318';

const exporter = new OTLPTraceExporter({
  url: `${otlpEndpoint}/v1/traces`,
});

const sdk = new NodeSDK({
  resource: new Resource({
    [ATTR_SERVICE_NAME]: serviceName,
  }),
  traceExporter: exporter,
  instrumentations: [
    // Automatically extracts trace context from incoming gRPC metadata
    // and injects it into outgoing calls — keeping the trace as one chain.
    new GrpcInstrumentation(),
  ],
});

sdk.start();

process.on('SIGTERM', () => {
  sdk.shutdown().finally(() => process.exit(0));
});

process.on('SIGINT', () => {
  sdk.shutdown().finally(() => process.exit(0));
});
