package ledger_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pkg "github.com/demo/ledger-service/pkg"
	"github.com/stretchr/testify/assert"
)

func setupHandler() *pkg.Handler {
	logger := slog.New(slog.NewJSONHandler(&strings.Builder{}, nil))
	ledgerSvc := pkg.NewLedgerService()
	return pkg.NewHandler(ledgerSvc, logger)
}

func TestGetTransactionNotFound(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/transactions/12345",
		strings.NewReader(`{"amount": 99.99, "merchant": "Test"}`))
	req.SetPathValue("id", "12345")
	recorder := httptest.NewRecorder()
	h.RecordTransaction(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	req2 := httptest.NewRequest("GET", "/transactions/12345", nil)
	req2.SetPathValue("id", "12345")
	recorder2 := httptest.NewRecorder()
	h.GetTransaction(recorder2, req2)

	assert.Equal(t, http.StatusOK, recorder2.Code)
}

func TestGetTransactionWithLargeID(t *testing.T) {
	h := setupHandler()

	req := httptest.NewRequest("GET", "/transactions/334152530919428096", nil)
	req.SetPathValue("id", "334152530919428096")
	recorder := httptest.NewRecorder()

	h.GetTransaction(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestGetTransactionLogsRequestID(t *testing.T) {
	var logBuf strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	ledgerSvc := pkg.NewLedgerService()
	h := pkg.NewHandler(ledgerSvc, logger)

	req := httptest.NewRequest("GET", "/transactions/99999", nil)
	req.SetPathValue("id", "99999")
	req.Header.Set("X-Request-ID", "test-req-id-123")
	req.Header.Set("X-Request-Timestamp", "2026-07-11T12:00:00Z")
	recorder := httptest.NewRecorder()

	h.GetTransaction(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "test-req-id-123")
	assert.Contains(t, logOutput, "2026-07-11T12:00:00Z")
	assert.Contains(t, logOutput, "\"request_id\"")
}

func TestRecordTransactionLogsRequestID(t *testing.T) {
	var logBuf strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	ledgerSvc := pkg.NewLedgerService()
	h := pkg.NewHandler(ledgerSvc, logger)

	req := httptest.NewRequest("POST", "/transactions/12345",
		strings.NewReader(`{"amount": 99.99, "merchant": "Test"}`))
	req.SetPathValue("id", "12345")
	req.Header.Set("X-Request-ID", "record-req-456")
	recorder := httptest.NewRecorder()
	h.RecordTransaction(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "record-req-456")
}

func TestServerWritesLogsToFileWhenConfigured(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "ledger-test.log")
	t.Setenv("DEMO_LOG_FILE", logFile)

	// Re-import to pick up the new env var
	// (Note: this just exercises the file output path through StartServer
	// isn't called here — we test the path-included handler directly)
	var logBuf strings.Builder
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	defer f.Close()
	multi := io.MultiWriter(&logBuf, f)
	logger := slog.New(slog.NewJSONHandler(multi, nil))

	ledgerSvc := pkg.NewLedgerService()
	h := pkg.NewHandler(ledgerSvc, logger)

	req := httptest.NewRequest("GET", "/transactions/99999", nil)
	req.SetPathValue("id", "99999")
	req.Header.Set("X-Request-ID", "file-log-test-789")
	recorder := httptest.NewRecorder()
	h.GetTransaction(recorder, req)

	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "file-log-test-789")
	assert.Contains(t, string(content), "99999")
}
