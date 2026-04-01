package tradingstrategy

import "time"

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

type AccountSnapshot struct {
	CashBalance      float64
	BuyingPower      float64
	PositionQuantity float64
	HasOpenOrder     bool
}

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
		Now:              snapshot.Now,
	}
}
