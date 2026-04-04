package replay

import (
	"context"
	"time"
)

type Strategy interface {
	Load(ctx context.Context, input LoadInput) (LoadOutput, error)
}

type LoadInput struct {
	Source     string
	Symbol     string
	Timeframe  string // canonical candle interval, e.g. "1Min", "1Hour", "1Day"
	Start      string // RFC 3339 start time (inclusive)
	End        string // RFC 3339 end time (inclusive), may be empty
	WarmupBars int
	Alpaca     AlpacaInput
	TastyTrade TastyTradeInput
}

type LoadOutput struct {
	Prices          []PricePoint
	IndicatorPrices []PricePoint
	Events          []Event
}

type AlpacaInput struct {
	Limit int
	Feed  string
}

type TastyTradeInput struct {
	BrokerType        string
	CollectionTimeout time.Duration
	MaxCandles        int
}
