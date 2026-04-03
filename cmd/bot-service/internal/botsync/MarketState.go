package botsync

import (
	"time"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

type sessionRange struct {
	bucket string
	open   float64
	high   float64
	low    float64
	ready  bool
}

type MarketState struct {
	symbol          string
	sessionInterval string
	session         sessionRange
	lastTradePrice  *float64
	bidPrice        *float64
	askPrice        *float64
	bidSize         *float64
	askSize         *float64
	dayVolume       *float64
	lastTradeSize   *float64
}

func NewMarketState(symbol string, sessionInterval string) *MarketState {
	return &MarketState{
		symbol:          symbol,
		sessionInterval: normalizeIndicatorResetInterval(sessionInterval),
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
	snapshotSessionOpen := state.session.open
	snapshotSessionHigh := state.session.high
	snapshotSessionLow := state.session.low
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
	bucket := indicatorResetBucket(now, state.sessionInterval)
	if state.session.bucket == bucket {
		return
	}
	state.session = sessionRange{bucket: bucket}
	state.dayVolume = nil
	state.lastTradeSize = nil
	state.lastTradePrice = nil
}

func (state *MarketState) updateSessionRange(price float64) {
	if !state.session.ready {
		state.session.open = price
		state.session.high = price
		state.session.low = price
		state.session.ready = true
		return
	}
	if price > state.session.high {
		state.session.high = price
	}
	if price < state.session.low {
		state.session.low = price
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
