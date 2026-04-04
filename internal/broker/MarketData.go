package broker

import (
	"context"
	"time"

	"github.com/kduong/trading-backend/internal/iterator"
)

type MarketDataType string

const (
	MarketDataTypeQuote  MarketDataType = "quote"
	MarketDataTypeTrade  MarketDataType = "trade"
	MarketDataTypeCandle MarketDataType = "candle"
)

type MarketDataMessage struct {
	Type       MarketDataType `json:"type"`
	Symbol     string         `json:"symbol"`
	Quote      *Quote         `json:"quote,omitempty"`
	Trade      *Trade         `json:"trade,omitempty"`
	Candle     *Candle        `json:"candle,omitempty"`
	ReceivedAt time.Time      `json:"received_at"`
}

type Quote struct {
	BidPrice float64 `json:"bidPrice"`
	AskPrice float64 `json:"askPrice"`
	BidSize  float64 `json:"bidSize"`
	AskSize  float64 `json:"askSize"`
}

type Trade struct {
	Price     float64  `json:"price"`
	DayVolume *float64 `json:"dayVolume,omitempty"`
	Size      *float64 `json:"size,omitempty"`
}

type Candle struct {
	Open         float64  `json:"open"`
	High         float64  `json:"high"`
	Low          float64  `json:"low"`
	Close        float64  `json:"close"`
	Volume       *float64 `json:"volume,omitempty"`
	OpenInterest *float64 `json:"openInterest,omitempty"`
}

type StreamMarketDataInput struct {
	Symbol string
}

type GetHistoricaDataInput struct {
	Symbol         string
	CandleInterval string
	FromTime       time.Time
}

type MarketDataClient interface {
	Stream(ctx context.Context, input StreamMarketDataInput) iterator.Iterator[*MarketDataMessage]
	GetHistoricalData(ctx context.Context, input GetHistoricaDataInput) iterator.Iterator[*MarketDataMessage]
}

type MarketDataClientFactory interface {
	Get(ctx context.Context, account *Account) MarketDataClient
}
