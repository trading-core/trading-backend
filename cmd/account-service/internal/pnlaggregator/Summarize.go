package pnlaggregator

import "github.com/kduong/trading-backend/internal/broker"

type Summary struct {
	TotalTrades     int     `json:"total_trades"`
	WinningTrades   int     `json:"winning_trades"`
	LosingTrades    int     `json:"losing_trades"`
	NetPnL          float64 `json:"net_pnl"`
	NetPnLAfterFees float64 `json:"net_pnl_after_fees"`
	Fees            float64 `json:"fees"`
	GrossWins       float64 `json:"gross_wins"`
	GrossLosses     float64 `json:"gross_losses"`
	WinRate         float64 `json:"win_rate"`
}

// Summarize counts closing trades as the unit of "trade" — every close in
// MatchRealizedPnL has a signed realized PnL, partitioning the window into
// wins (realized > 0), losses (realized < 0), and scratches (realized == 0,
// excluded from both counts). Fees are summed across every transaction in the
// window so non-trade rows (cash movements, dividends) still drag on net PnL.
func Summarize(transactions []broker.Transaction) Summary {
	summary := Summary{}
	for _, transaction := range transactions {
		summary.Fees += transaction.Fees
		if transaction.Type != "Trade" || transaction.Effect != broker.OrderEffectClose {
			continue
		}
		summary.TotalTrades++
		summary.NetPnL += transaction.RealizedPnL
		switch {
		case transaction.RealizedPnL > 0:
			summary.WinningTrades++
			summary.GrossWins += transaction.RealizedPnL
		case transaction.RealizedPnL < 0:
			summary.LosingTrades++
			summary.GrossLosses += transaction.RealizedPnL
		}
	}
	summary.NetPnLAfterFees = summary.NetPnL - summary.Fees
	decided := summary.WinningTrades + summary.LosingTrades
	if decided > 0 {
		summary.WinRate = float64(summary.WinningTrades) / float64(decided)
	}
	return summary
}
