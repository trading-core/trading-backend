package pnlaggregator

import (
	"sort"
	"time"

	"github.com/kduong/trading-backend/internal/broker"
)

type DailyPnL struct {
	Date        string  `json:"date"`
	RealizedPnL float64 `json:"realized_pnl"`
	TradeCount  int     `json:"trade_count"`
	Fees        float64 `json:"fees"`
}

type Result struct {
	Days []DailyPnL `json:"days"`
}

// Aggregate buckets transactions by UTC calendar date, summing realized PnL,
// fees, and trade counts. Non-trade transactions (cash movements, dividends)
// contribute only fees, not trade counts.
func Aggregate(transactions []broker.Transaction) *Result {
	dayByDate := make(map[string]*DailyPnL)
	for _, transaction := range transactions {
		date := extractUTCDate(transaction.ExecutedAt)
		if date == "" {
			continue
		}
		day, exists := dayByDate[date]
		if !exists {
			day = &DailyPnL{Date: date}
			dayByDate[date] = day
		}
		day.RealizedPnL += transaction.RealizedPnL
		day.Fees += transaction.Fees
		if transaction.Type == "Trade" {
			day.TradeCount++
		}
	}
	days := make([]DailyPnL, 0, len(dayByDate))
	for _, day := range dayByDate {
		days = append(days, *day)
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].Date < days[j].Date
	})
	return &Result{Days: days}
}

func extractUTCDate(executedAt string) string {
	parsed, err := time.Parse(time.RFC3339, executedAt)
	if err != nil {
		return ""
	}
	return parsed.UTC().Format("2006-01-02")
}
