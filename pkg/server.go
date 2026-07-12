package ledger

import (
	"log/slog"
	"net/http"
	"os"
)

func StartServer() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ledgerSvc := NewLedgerService()
	handler := NewHandler(ledgerSvc, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /transactions/{id}", handler.GetTransaction)
	mux.HandleFunc("POST /transactions/{id}", handler.RecordTransaction)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "service": "ledger"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	logger.Info("starting ledger service", "port", port)
	return http.ListenAndServe(":"+port, mux)
}
