package ledger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTransaction_NotFound(t *testing.T) {
	ledger := NewLedgerService()
	handler := NewHandler(ledger, nil)

	req := httptest.NewRequest("GET", "/transactions/99999", nil)
	w := httptest.NewRecorder()

	handler.GetTransaction(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRecordAndLookup(t *testing.T) {
	ledger := NewLedgerService()
	handler := NewHandler(ledger, nil)

	// Record a transaction
	body, _ := json.Marshal(map[string]float64{"amount": 100.0})
	req := httptest.NewRequest("POST", "/transactions/12345", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.RecordTransaction(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Lookup the transaction
	req = httptest.NewRequest("GET", "/transactions/12345", nil)
	w = httptest.NewRecorder()

	handler.GetTransaction(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var txn Transaction
	json.Unmarshal(w.Body.Bytes(), &txn)
	assert.Equal(t, int64(12345), txn.ID)
	assert.Equal(t, 100.0, txn.Amount)
}

func TestGetTransaction_InvalidID(t *testing.T) {
	ledger := NewLedgerService()
	handler := NewHandler(ledger, nil)

	req := httptest.NewRequest("GET", "/transactions/not-a-number", nil)
	w := httptest.NewRecorder()

	handler.GetTransaction(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
