package tradingstrategy

import "time"

// MarketSnapshot is the normalized view of the latest market state for a symbol.
//
// Pointer fields are optional because a given market data tick may not include
// every quote or trade attribute.
type MarketSnapshot struct {
	Symbol           string
	LastTradePrice   *float64
	BidPrice         *float64
	AskPrice         *float64
	BidSize          *float64
	AskSize          *float64
	DayVolume        *float64
	LastTradeSize    *float64
	SessionOpenPrice float64
	SessionHighPrice float64
	SessionLowPrice  float64
	Now              time.Time
}

// AccountSnapshot contains the account state relevant to trading decisions.
type AccountSnapshot struct {
	CashBalance      float64
	BuyingPower      float64
	PositionQuantity float64
	HasOpenOrder     bool
	EntryPrice       float64
	HighSinceEntry   float64 // highest price reached since the current position was opened; used by ATRStopStrategy for trailing stop
}

// NewEvaluateInput combines market and account snapshots into the single input
// consumed by strategy evaluation.
//
// Price is derived from the best available market signal in this order:
// last trade, mid price, bid, then ask. Spread is only set when both bid and
// ask are present.
func NewEvaluateInput(snapshot MarketSnapshot, account AccountSnapshot) EvaluateInput {
	var spread *float64
	if snapshot.BidPrice != nil && snapshot.AskPrice != nil {
		value := *snapshot.AskPrice - *snapshot.BidPrice
		spread = &value
	}
	price := 0.0
	switch {
	case snapshot.LastTradePrice != nil:
		price = *snapshot.LastTradePrice
	case snapshot.BidPrice != nil && snapshot.AskPrice != nil:
		price = (*snapshot.BidPrice + *snapshot.AskPrice) / 2
	case snapshot.BidPrice != nil:
		price = *snapshot.BidPrice
	case snapshot.AskPrice != nil:
		price = *snapshot.AskPrice
	}
	return EvaluateInput{
		Price:            price,
		LastTradePrice:   snapshot.LastTradePrice,
		BidPrice:         snapshot.BidPrice,
		AskPrice:         snapshot.AskPrice,
		BidSize:          snapshot.BidSize,
		AskSize:          snapshot.AskSize,
		Spread:           spread,
		DayVolume:        snapshot.DayVolume,
		LastTradeSize:    snapshot.LastTradeSize,
		SessionOpenPrice: snapshot.SessionOpenPrice,
		SessionHighPrice: snapshot.SessionHighPrice,
		SessionLowPrice:  snapshot.SessionLowPrice,
		CashBalance:      account.CashBalance,
		BuyingPower:      account.BuyingPower,
		PositionQuantity: account.PositionQuantity,
		HasOpenOrder:     account.HasOpenOrder,
		EntryPrice:       account.EntryPrice,
		HighSinceEntry:   account.HighSinceEntry,
		Now:              snapshot.Now,
	}
}
