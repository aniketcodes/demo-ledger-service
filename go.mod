module github.com/demo/ledger-service

go 1.22

require (
	github.com/prometheus/client_golang v1.19.0
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.0
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
	go.opentelemetry.io/otel/sdk v1.21.0
)
