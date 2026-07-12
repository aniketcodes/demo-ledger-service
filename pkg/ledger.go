package ledger

import (
	"context"
	"sync"
	"time"
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
	storeMu.RLock()
	defer storeMu.RUnlock()

	txn, ok := store[txnID]
	if !ok {
		return nil, nil
	}

	return txn, nil
}
