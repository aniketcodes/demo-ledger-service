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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid transaction ID"})
		return
	}

	txn, err := h.ledger.GetTransaction(r.Context(), txnID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if txn == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "transaction not found"})
		return
	}

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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
