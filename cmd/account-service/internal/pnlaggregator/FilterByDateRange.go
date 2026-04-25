package pnlaggregator

import (
	"github.com/kduong/trading-backend/internal/broker"
)

// FilterByDateRange keeps only transactions whose ExecutedAt UTC date lies
// within [fromDate, toDate] inclusive. Both bounds are YYYY-MM-DD. Rows with
// unparseable timestamps are dropped, mirroring Aggregate.
func FilterByDateRange(transactions []broker.Transaction, fromDate, toDate string) []broker.Transaction {
	filtered := make([]broker.Transaction, 0, len(transactions))
	for _, transaction := range transactions {
		date := extractUTCDate(transaction.ExecutedAt)
		if date == "" {
			continue
		}
		if date < fromDate || date > toDate {
			continue
		}
		filtered = append(filtered, transaction)
	}
	return filtered
}
