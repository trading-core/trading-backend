package replay

import "time"

type EventType string

const (
	EventTypeQuote EventType = "quote"
	EventTypeTrade EventType = "trade"
)

type Event struct {
	Type      EventType
	At        time.Time
	Symbol    string
	Trade     *Trade
	Quote     *Quote
	DayVolume *float64
	Size      *float64
}

type Trade struct {
	Price float64
}

type Quote struct {
	BidPrice float64
	AskPrice float64
	BidSize  float64
	AskSize  float64
}

type PricePoint struct {
	At    time.Time
	Close float64
}
