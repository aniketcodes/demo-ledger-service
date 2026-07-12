package ledger

import (
	"io"
	"log/slog"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func StartServer() error {
	// Compose stdout + optional file output so Promtail can scrape the file
	var writer io.Writer = os.Stdout
	if logFile := os.Getenv("DEMO_LOG_FILE"); logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			writer = io.MultiWriter(os.Stdout, f)
		}
	}

	logger := slog.New(slog.NewJSONHandler(writer, nil))
	slog.SetDefault(logger)

	ledgerSvc := NewLedgerService()
	handler := NewHandler(ledgerSvc, logger)

	shutdown, err := SetupTracing()
	if err != nil {
		logger.Error("failed to setup tracing", "error", err)
	} else {
		defer shutdown()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /transactions/{id}", handler.GetTransaction)
	mux.HandleFunc("POST /transactions/{id}", handler.RecordTransaction)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "service": "ledger"})
	})
	mux.Handle("GET /metrics", MetricsHandler())

	otelHandler := otelhttp.NewHandler(mux, "ledger")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	logger.Info("starting ledger service", "port", port)
	return http.ListenAndServe(":"+port, otelHandler)
}
