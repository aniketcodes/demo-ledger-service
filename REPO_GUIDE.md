# demo-ledger-service — REPO GUIDE

## Overview

Go HTTP service that maintains the ledger. Receives transaction IDs from the order service and looks them up in an in-memory store. This service is **correct** — Go's `int64` handles 17-18 digit IDs natively. But it receives the **wrong ID** (corrupted by the TS order service) and returns "not found".

## Critical Fix Notes

- **C1:** Dropped `mattn/go-sqlite3` (CGO-only) → use in-memory `map[int64]*Transaction` with `sync.RWMutex`. No CGO needed, `CGO_ENABLED=0` works, smaller Alpine image.
- **C2:** Fixed build target: `go build -o /ledger .` (root `main.go`), not `./pkg/`.
- **C3:** Replaced `conn.Execute()` with in-memory map (no SQL at all — moot).
- **C4:** Fixed tests to instantiate `Handler`, import `testify/assert` + `strings`.
- **C10:** Wrapped mux with `otelhttp.NewHandler()` to extract `traceparent` from incoming requests. Without this, the trace breaks at TS→Go boundary.
- **M5:** Dropped SQLite entirely — in-memory store is cleaner, no CGO issues, smaller image, and the narrative ("Go handles big ints natively") is better served.

## Tech Stack

- **Language:** Go 1.22
- **Framework:** `net/http` (standard library, Go 1.22 `ServeMux` patterns)
- **DB:** In-memory `map[int64]*Transaction` with `sync.RWMutex` (no SQLite, no CGO)
- **Tracing:** OpenTelemetry (OTLP/gRPC) + `otelhttp` for HTTP server instrumentation
- **Metrics:** Prometheus (`prometheus/client_golang`)
- **Logging:** Structured JSON (`log/slog`)
- **Container:** Single Docker image via `docker compose up`

## The Bug Role

This service is the **victim**. It receives a corrupted transaction ID from the TS order service. It looks up the ID correctly, finds nothing, and returns "transaction not found". The code is correct — the data it receives is wrong.

**Why it's correct:** Go's `int64` can represent integers up to `9223372036854775807` (about 19 digits). Any 17-18 digit ID fits comfortably. Go's `encoding/json` preserves full precision when parsing JSON numbers into `int64`.

**The irony:** The Go service CAN handle large IDs. It's the TypeScript service that can't. The Go service reports "not found" correctly — the transaction with the corrupted ID genuinely doesn't exist.

## Directory Structure

```
demo-ledger-service/
├── REPO_GUIDE.md            ← you are here
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── .env.example
├── main.go                  # Entry point — package main
├── pkg/
│   ├── handler.go           # HTTP handler — looks up txn by ID
│   ├── ledger.go            # Business logic (in-memory store)
│   ├── tracing.go           # OpenTelemetry setup
│   ├── metrics.go           # Prometheus metrics
│   └── server.go            # HTTP server setup (with otelhttp)
├── tests/
│   ├── handler_test.go      # Unit tests (small IDs, real Handler)
│   └── integration_test.go  # Integration tests
└── README.md
```

## File-by-File Specification

### `Dockerfile`

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# C2 fix: build from root (main.go is here), not ./pkg/
# C1 fix: CGO_ENABLED=0 works because we use in-memory map (no SQLite)
RUN CGO_ENABLED=0 go build -o /ledger .

FROM alpine:latest
COPY --from=builder /ledger /ledger

EXPOSE 8003
CMD ["/ledger"]
```

> **C1 fix:** No SQLite dep → no CGO → `CGO_ENABLED=0` works → tiny Alpine image, no `sqlite-libs`.
> **C2 fix:** `go build -o /ledger .` (root package with `main.go`).

### `docker-compose.yml`

```yaml
services:
  ledger-service:
    build: .
    ports:
      - "8003:8003"
    environment:
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
      - OTEL_SERVICE_NAME=demo-ledger-service
    networks:
      - demo-network

networks:
  demo-network:
    external: true
```

### `go.mod`

```go
module github.com/demo/ledger-service

go 1.22

require (
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
    go.opentelemetry.io/otel/propagation v1.21.0
    go.opentelemetry.io/otel/sdk v1.21.0
    go.opentelemetry.io/otel/trace v1.21.0
    go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.0
    github.com/prometheus/client_golang v1.19.0
    github.com/stretchr/testify v1.8.4
)
```

> **C10 fix:** Added `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`.
> **C4 fix:** Added `github.com/stretchr/testify` for test assertions.

### `pkg/ledger.go`

```go
package ledger

import (
    "context"
    "sync"
    "time"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

// In-memory store — no SQLite, no CGO
var (
    store   = make(map[int64]*Transaction)
    storeMu sync.RWMutex
)

type Transaction struct {
    ID        int64   `json:"transaction_id"`
    Amount    float64 `json:"amount"`
    Status    string  `json:"status"`
    CreatedAt string  `json:"created_at"`
}

type LedgerService struct{}

func NewLedgerService() *LedgerService {
    return &LedgerService{}
}

func (s *LedgerService) RecordTransaction(ctx context.Context, txnID int64, amount float64) error {
    tracer := otel.Tracer("demo-ledger-service")
    ctx, span := tracer.Start(ctx, "ledger.record_transaction")
    defer span.End()

    span.SetAttributes(
        attribute.Int64("txn_id", txnID),
        attribute.Float64("amount", amount),
    )

    storeMu.Lock()
    defer storeMu.Unlock()
    store[txnID] = &Transaction{
        ID:        txnID,
        Amount:    amount,
        Status:    "recorded",
        CreatedAt: time.Now().Format(time.RFC3339),
    }
    return nil
}

func (s *LedgerService) GetTransaction(ctx context.Context, txnID int64) (*Transaction, error) {
    tracer := otel.Tracer("demo-ledger-service")
    ctx, span := tracer.Start(ctx, "ledger.lookup_transaction")
    defer span.End()

    span.SetAttributes(attribute.Int64("txn_id", txnID))

    storeMu.RLock()
    defer storeMu.RUnlock()

    txn, ok := store[txnID]
    if !ok {
        span.SetAttributes(
            attribute.Bool("found", false),
            attribute.String("error", "transaction not found"),
        )
        return nil, nil
    }

    span.SetAttributes(attribute.Bool("found", true))
    return txn, nil
}
```

> **M5 fix:** In-memory `map[int64]*Transaction` with `sync.RWMutex`. No SQLite. No CGO.

### `pkg/handler.go`

```go
package ledger

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "strconv"
)

type Handler struct {
    ledger *LedgerService
    logger *slog.Logger
}

func NewHandler(ledger *LedgerService, logger *slog.Logger) *Handler {
    return &Handler{ledger: ledger, logger: logger}
}

func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
    txnIDStr := r.PathValue("id")
    txnID, err := strconv.ParseInt(txnIDStr, 10, 64)
    if err != nil {
        h.logger.Error("invalid transaction ID",
            "txn_id_str", txnIDStr,
            "error", err.Error(),
        )
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid transaction ID"})
        return
    }

    h.logger.Info("looking up transaction",
        "txn_id", txnID,
        "service", "ledger",
    )

    txn, err := h.ledger.GetTransaction(r.Context(), txnID)
    if err != nil {
        h.logger.Error("lookup failed",
            "txn_id", txnID,
            "error", err.Error(),
        )
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
        return
    }

    if txn == nil {
        // B4 fix: increment the "not found" counter so the Error Rate dashboard shows the spike
        TransactionsNotFound.Inc()
        h.logger.Info("transaction not found",
            "txn_id", txnID,
            "service", "ledger",
        )
        writeJSON(w, http.StatusNotFound, map[string]string{"error": "transaction not found"})
        return
    }

    // B4 fix: increment the "found" counter
    TransactionsFound.Inc()
    h.logger.Info("transaction found",
        "txn_id", txnID,
        "amount", txn.Amount,
        "status", txn.Status,
        "service", "ledger",
    )
    writeJSON(w, http.StatusOK, txn)
}

func (h *Handler) RecordTransaction(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Amount   float64 `json:"amount"`
        Merchant string  `json:"merchant"`
    }

    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
        return
    }

    txnIDStr := r.PathValue("id")
    txnID, err := strconv.ParseInt(txnIDStr, 10, 64)
    if err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid transaction ID"})
        return
    }

    if err := h.ledger.RecordTransaction(r.Context(), txnID, body.Amount); err != nil {
        h.logger.Error("failed to record transaction",
            "txn_id", txnID,
            "error", err.Error(),
        )
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record"})
        return
    }

    h.logger.Info("transaction recorded",
        "txn_id", txnID,
        "amount", body.Amount,
        "service", "ledger",
    )

    writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}
```

### `pkg/tracing.go`

```go
package ledger

import (
    "context"
    "fmt"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func SetupTracing() (func(), error) {
    endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    if endpoint == "" {
        endpoint = "localhost:4317"
    }

    serviceName := os.Getenv("OTEL_SERVICE_NAME")
    if serviceName == "" {
        serviceName = "demo-ledger-service"
    }

    exporter, err := otlptracegrpc.New(context.Background(),
        otlptracegrpc.WithEndpoint(endpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create exporter: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(serviceName),
        )),
    )

    otel.SetTracerProvider(tp)

    // CRITICAL: Set the propagator to TraceContext + Baggage.
    // The default propagator is noop — WITHOUT this, otelhttp.NewHandler()
    // won't extract the incoming traceparent header from the TS service.
    // The Go span would be a root span, disconnected from the TS span.
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return func() {
        tp.Shutdown(context.Background())
    }, nil
}
```

### `pkg/metrics.go`

```go
package ledger

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

var (
    TransactionsFound = promauto.NewCounter(prometheus.CounterOpts{
        Name: "ledger_transactions_found_total",
        Help: "Total transactions found in store",
    })

    TransactionsNotFound = promauto.NewCounter(prometheus.CounterOpts{
        Name: "ledger_transactions_not_found_total",
        Help: "Total transactions not found in store",
    })
)

func MetricsHandler() http.Handler {
    return promhttp.Handler()
}
```

### `pkg/server.go`

```go
package ledger

import (
    "log/slog"
    "net/http"
    "os"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func StartServer() error {
    // Structured JSON logging
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    // Initialize services
    ledgerSvc := NewLedgerService()
    handler := NewHandler(ledgerSvc, logger)

    // Setup tracing
    shutdown, err := SetupTracing()
    if err != nil {
        logger.Error("failed to setup tracing", "error", err)
    } else {
        defer shutdown()
    }

    // Routes (Go 1.22 ServeMux patterns)
    mux := http.NewServeMux()
    mux.HandleFunc("GET /transactions/{id}", handler.GetTransaction)
    mux.HandleFunc("POST /transactions/{id}", handler.RecordTransaction)
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, 200, map[string]string{"status": "ok", "service": "ledger"})
    })
    mux.Handle("GET /metrics", MetricsHandler())

    // C10 fix: Wrap mux with otelhttp so incoming traceparent headers
    // are extracted and the trace continues from the TS service.
    // Without this, every Go request is a root span (no parent).
    otelHandler := otelhttp.NewHandler(mux, "ledger")

    port := os.Getenv("PORT")
    if port == "" {
        port = "8003"
    }

    logger.Info("starting ledger service", "port", port)
    return http.ListenAndServe(":"+port, otelHandler)
}
```

> **C10 fix:** `otelhttp.NewHandler(mux, "ledger")` wraps the mux. Incoming `traceparent` header from TS is extracted. Trace continues across the TS→Go boundary.

### `main.go` (root)

```go
package main

import "github.com/demo/ledger-service/pkg"

func main() {
    if err := ledger.StartServer(); err != nil {
        panic(err)
    }
}
```

> **C2 fix:** `main.go` at repo root with `package main`. `go build -o /ledger .` from root.

### `tests/handler_test.go`

```go
package ledger_test

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/demo/ledger-service/pkg"
    "github.com/stretchr/testify/assert"
)

// C4 fix: Instantiates a real Handler, uses testify/assert, imports strings

func setupHandler() *pkg.Handler {
    logger := slog.New(slog.NewJSONHandler(&strings.Builder{}, nil))
    ledgerSvc := pkg.NewLedgerService()
    return pkg.NewHandler(ledgerSvc, logger)
}

func TestGetTransactionNotFound(t *testing.T) {
    // Use a unique ID that doesn't exist in the store
    h := setupHandler()

    req := httptest.NewRequest("GET", "/transactions/99999", nil)
    req.SetPathValue("id", "99999")
    recorder := httptest.NewRecorder()

    h.GetTransaction(recorder, req)

    assert.Equal(t, http.StatusNotFound, recorder.Code)

    var body map[string]string
    json.Unmarshal(recorder.Body.Bytes(), &body)
    assert.Equal(t, "transaction not found", body["error"])
}

func TestRecordAndGetTransaction(t *testing.T) {
    h := setupHandler()

    // Record a transaction with a small ID
    req := httptest.NewRequest("POST", "/transactions/12345",
        strings.NewReader(`{"amount": 99.99, "merchant": "Test"}`))
    req.SetPathValue("id", "12345")
    recorder := httptest.NewRecorder()
    h.RecordTransaction(recorder, req)

    assert.Equal(t, http.StatusOK, recorder.Code)

    // Now look it up — should find it
    req2 := httptest.NewRequest("GET", "/transactions/12345", nil)
    req2.SetPathValue("id", "12345")
    recorder2 := httptest.NewRecorder()
    h.GetTransaction(recorder2, req2)

    assert.Equal(t, http.StatusOK, recorder2.Code)
    // PASSES — small ID, no precision issue
}

func TestGetTransactionWithLargeID(t *testing.T) {
    // Go handles large IDs correctly — no precision loss
    h := setupHandler()

    req := httptest.NewRequest("GET", "/transactions/334152530919428096", nil)
    req.SetPathValue("id", "334152530919428096")
    recorder := httptest.NewRecorder()

    h.GetTransaction(recorder, req)

    // Returns 404 — correct! This transaction doesn't exist in the store
    assert.Equal(t, http.StatusNotFound, recorder.Code)
}

// NOTE: All tests use small IDs or test Go in isolation.
// No test sends a large ID FROM TypeScript through JSON.parse().
// THAT's where the precision loss happens, and no test covers it.
```

> **C4 fix:** Creates real `pkg.Handler` via `setupHandler()`. Imports `testify/assert`, `strings`, `encoding/json`. All references are valid.

## Build & Run

```bash
# Create the shared network (first time only)
docker network create demo-network

# Start
docker compose up

# Or via umbrella compose:
# docker compose up -d

# Test
go test ./tests/... -v

# Manual test
curl http://localhost:8003/health
curl http://localhost:8003/transactions/12345
```

## Key Config

| Env Var | Default | Description |
|---------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTel collector (host:port, no scheme) |
| `OTEL_SERVICE_NAME` | `demo-ledger-service` | Service name in traces |
| `PORT` | `8003` | HTTP port |

## Observability

- **Traces:** span `ledger.lookup_transaction` with attributes `txn_id` (corrupted — received wrong ID from TS), `found: false`. Trace receives `traceparent` from TS via `otelhttp.NewHandler()`.
- **Metrics:** `ledger_transactions_found_total`, `ledger_transactions_not_found_total`
- **Logs:** `slog` JSON structured, include `txn_id` field (also corrupted — the wrong ID from TS)