package tastytrade

import "time"

type MessageEventType string

const (
	MessageEventTypeQuote MessageEventType = "Quote"
	MessageEventTypeTrade MessageEventType = "Trade"
)

type MessageEvent struct {
	Type  MessageEventType   `json:"type"`
	Quote *MessageEventQuote `json:"quote,omitempty"`
	Trade *MessageEventTrade `json:"trade,omitempty"`
}

type MessageEventQuote struct {
	EventSymbol string     `json:"eventSymbol"`
	BidPrice    float64    `json:"bidPrice"`
	AskPrice    float64    `json:"askPrice"`
	BidSize     float64    `json:"bidSize"`
	AskSize     float64    `json:"askSize"`
	EventTime   *time.Time `json:"eventTime,omitempty"`
}

type MessageEventTrade struct {
	EventSymbol string     `json:"eventSymbol"`
	Price       float64    `json:"price"`
	DayVolume   *float64   `json:"dayVolume,omitempty"`
	Size        *float64   `json:"size,omitempty"`
	EventTime   *time.Time `json:"eventTime,omitempty"`
}
