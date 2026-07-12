# Demo Ledger Service

Go service for managing transactions in the demo microservices architecture.

## Features
- In-memory transaction store
- HTTP API for transaction operations
- OpenTelemetry distributed tracing
- Prometheus metrics collection
- Structured JSON logging

## Endpoints
- `GET /transactions/{id}` - Lookup a transaction
- `POST /transactions/{id}` - Record a transaction
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

## Configuration
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTel collector endpoint (default: localhost:4317)
- `OTEL_SERVICE_NAME` - Service name for tracing (default: demo-ledger-service)
- `DEMO_LOG_FILE` - Log file path for Promtail scraping
- `PORT` - Server port (default: 8003)

## Known Issues
- **In-memory store**: Transaction data is lost on service restart
- **Large ID handling**: Go correctly handles int64 IDs, but may receive corrupted values from JavaScript services
