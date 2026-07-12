package ledger

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	TransactionsFound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledger_transactions_found_total",
		Help: "Total transactions found in store",
	})

	TransactionsNotFound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledger_transactions_not_found_total",
		Help: "Total transactions not found in store",
	})
)

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
