package ledger

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

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
