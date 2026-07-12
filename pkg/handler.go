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

func requestContext(r *http.Request) (requestID, timestamp string) {
	requestID = r.Header.Get("X-Request-ID")
	timestamp = r.Header.Get("X-Request-Timestamp")
	return
}

func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	requestID, timestamp := requestContext(r)

	txnIDStr := r.PathValue("id")
	txnID, err := strconv.ParseInt(txnIDStr, 10, 64)
	if err != nil {
		h.logger.Error("invalid transaction ID",
			"request_id", requestID,
			"timestamp", timestamp,
			"txn_id_str", txnIDStr,
			"error", err.Error(),
		)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid transaction ID"})
		return
	}

	h.logger.Info("looking up transaction",
		"request_id", requestID,
		"timestamp", timestamp,
		"txn_id", txnID,
		"service", "ledger",
	)

	txn, err := h.ledger.GetTransaction(r.Context(), txnID)
	if err != nil {
		h.logger.Error("lookup failed",
			"request_id", requestID,
			"timestamp", timestamp,
			"txn_id", txnID,
			"error", err.Error(),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if txn == nil {
		h.logger.Info("transaction not found",
			"request_id", requestID,
			"timestamp", timestamp,
			"txn_id", txnID,
			"service", "ledger",
		)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "transaction not found"})
		return
	}

	h.logger.Info("transaction found",
		"request_id", requestID,
		"timestamp", timestamp,
		"txn_id", txnID,
		"amount", txn.Amount,
		"status", txn.Status,
		"service", "ledger",
	)
	writeJSON(w, http.StatusOK, txn)
}

func (h *Handler) RecordTransaction(w http.ResponseWriter, r *http.Request) {
	requestID, timestamp := requestContext(r)

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
			"request_id", requestID,
			"timestamp", timestamp,
			"txn_id", txnID,
			"error", err.Error(),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record"})
		return
	}

	h.logger.Info("transaction recorded",
		"request_id", requestID,
		"timestamp", timestamp,
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
