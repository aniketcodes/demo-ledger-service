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
