package replay

import (
	"time"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

type State struct {
	symbol         string
	session        sessionRange
	lastTradePrice *float64
	bidPrice       *float64
	askPrice       *float64
	bidSize        *float64
	askSize        *float64
	dayVolume      *float64
	lastTradeSize  *float64
}

type sessionRange struct {
	date  string
	open  float64
	high  float64
	low   float64
	ready bool
}

func NewState(symbol string) *State {
	return &State{symbol: symbol}
}

func (state *State) Apply(event Event) tradingstrategy.MarketSnapshot {
	if event.Symbol != "" {
		state.symbol = event.Symbol
	}
	state.resetSessionIfNeeded(event.At)
	snapshotSessionOpen := state.session.open
	snapshotSessionHigh := state.session.high
	snapshotSessionLow := state.session.low
	if event.Type == EventTypeQuote && event.Quote != nil {
		state.bidPrice = float64Ptr(event.Quote.BidPrice)
		state.askPrice = float64Ptr(event.Quote.AskPrice)
		state.bidSize = float64Ptr(event.Quote.BidSize)
		state.askSize = float64Ptr(event.Quote.AskSize)
	}
	if event.Type == EventTypeTrade && event.Trade != nil {
		state.lastTradePrice = float64Ptr(event.Trade.Price)
		state.dayVolume = cloneFloat64Ptr(event.DayVolume)
		state.lastTradeSize = cloneFloat64Ptr(event.Size)
		state.updateSessionRange(event.Trade.Price)
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
		Now:              event.At,
	}
}

// LastSnapshot returns the current state as a snapshot without advancing time
// or re-applying any event. Use this to retrieve state after all events have
// been processed, avoiding a redundant Apply that would double-update session
// tracking.
func (state *State) LastSnapshot() tradingstrategy.MarketSnapshot {
	return tradingstrategy.MarketSnapshot{
		Symbol:           state.symbol,
		LastTradePrice:   cloneFloat64Ptr(state.lastTradePrice),
		BidPrice:         cloneFloat64Ptr(state.bidPrice),
		AskPrice:         cloneFloat64Ptr(state.askPrice),
		BidSize:          cloneFloat64Ptr(state.bidSize),
		AskSize:          cloneFloat64Ptr(state.askSize),
		DayVolume:        cloneFloat64Ptr(state.dayVolume),
		LastTradeSize:    cloneFloat64Ptr(state.lastTradeSize),
		SessionOpenPrice: state.session.open,
		SessionHighPrice: state.session.high,
		SessionLowPrice:  state.session.low,
	}
}

func (state *State) resetSessionIfNeeded(now time.Time) {
	date := now.In(tradingstrategy.USMarketLocation).Format("2006-01-02")
	if state.session.date == date {
		return
	}
	state.session = sessionRange{date: date}
	state.dayVolume = nil
	state.lastTradeSize = nil
	state.lastTradePrice = nil
}

func (state *State) updateSessionRange(price float64) {
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
