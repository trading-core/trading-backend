package botsync

import (
	"time"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"
	"github.com/kduong/trading-backend/internal/broker"
)

type MarketState struct {
	symbol           string
	sessionDate      string
	lastTradePrice   *float64
	bidPrice         *float64
	askPrice         *float64
	bidSize          *float64
	askSize          *float64
	dayVolume        *float64
	lastTradeSize    *float64
	sessionOpenPrice float64
	sessionHighPrice float64
	sessionLowPrice  float64
}

func NewMarketState(symbol string) *MarketState {
	return &MarketState{
		symbol: symbol,
	}
}

func (state *MarketState) Apply(message *broker.MarketDataMessage) tradingstrategy.MarketSnapshot {
	now := time.Now()
	if message != nil && !message.ReceivedAt.IsZero() {
		now = message.ReceivedAt
	}
	state.resetSessionIfNeeded(now)
	// Capture session range before any trade update so the strategy evaluates
	// against the pre-tick high/low (prevents breakout condition from being
	// immediately false on the very tick that sets a new session high).
	snapshotSessionOpen := state.sessionOpenPrice
	snapshotSessionHigh := state.sessionHighPrice
	snapshotSessionLow := state.sessionLowPrice
	state.symbol = message.Symbol
	switch message.Type {
	case broker.MarketDataTypeQuote:
		state.bidPrice = float64Ptr(message.Quote.BidPrice)
		state.askPrice = float64Ptr(message.Quote.AskPrice)
		state.bidSize = float64Ptr(message.Quote.BidSize)
		state.askSize = float64Ptr(message.Quote.AskSize)
	case broker.MarketDataTypeTrade:
		state.lastTradePrice = float64Ptr(message.Trade.Price)
		state.dayVolume = cloneFloat64Ptr(message.Trade.DayVolume)
		state.lastTradeSize = cloneFloat64Ptr(message.Trade.Size)
		state.updateSessionRange(message.Trade.Price)
	}
	return tradingstrategy.MarketSnapshot{
		Symbol:           state.symbol,
		LastTradePrice:   cloneFloat64Ptr(state.lastTradePrice),
		BidPrice:         cloneFloat64Ptr(state.bidPrice),
		AskPrice:         cloneFloat64Ptr(state.askPrice),
		BidSize:          cloneFloat64Ptr(state.bidSize),
		AskSize:          cloneFloat64Ptr(state.askSize),
		DayVolume:        cloneFloat64Ptr(state.dayVolume),
		LastTradeSize:    cloneFloat64Ptr(state.lastTradeSize),
		SessionOpenPrice: snapshotSessionOpen,
		SessionHighPrice: snapshotSessionHigh,
		SessionLowPrice:  snapshotSessionLow,
		Now:              now,
	}
}

func (state *MarketState) resetSessionIfNeeded(now time.Time) {
	date := now.Format("2006-01-02")
	if state.sessionDate == date {
		return
	}
	state.sessionDate = date
	state.sessionOpenPrice = 0
	state.sessionHighPrice = 0
	state.sessionLowPrice = 0
	state.dayVolume = nil
	state.lastTradeSize = nil
	state.lastTradePrice = nil
}

func (state *MarketState) updateSessionRange(price float64) {
	if state.sessionOpenPrice == 0 {
		state.sessionOpenPrice = price
		state.sessionHighPrice = price
		state.sessionLowPrice = price
		return
	}
	if price > state.sessionHighPrice {
		state.sessionHighPrice = price
	}
	if state.sessionLowPrice == 0 || price < state.sessionLowPrice {
		state.sessionLowPrice = price
	}
}

func (state *MarketState) Symbol() string {
	return state.symbol
}

func float64Ptr(value float64) *float64 {
	return &value
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
