package ledger_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkg "github.com/demo/ledger-service/pkg"
	"github.com/stretchr/testify/assert"
)

func setupTestServer() *httptest.Server {
	logger := slog.New(slog.NewJSONHandler(&strings.Builder{}, nil))
	ledgerSvc := pkg.NewLedgerService()
	handler := pkg.NewHandler(ledgerSvc, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /transactions/{id}", handler.GetTransaction)
	mux.HandleFunc("POST /transactions/{id}", handler.RecordTransaction)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"ledger"}`))
	})

	return httptest.NewServer(mux)
}

func TestIntegrationHealthCheck(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
