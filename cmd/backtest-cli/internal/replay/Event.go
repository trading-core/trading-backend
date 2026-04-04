package replay

import (
	"sort"
	"time"
)

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

func EventsFromCandles(symbol string, candles []PricePoint) []Event {
	events := make([]Event, 0, len(candles))
	for _, c := range candles {
		events = append(events, Event{
			Type:   EventTypeTrade,
			At:     c.At,
			Symbol: symbol,
			Trade: &Trade{
				Price: c.Close,
			},
		})
	}
	return events
}

func PriceSeries(events []Event) []PricePoint {
	series := make([]PricePoint, 0, len(events))
	for _, event := range events {
		var price float64
		switch {
		case event.Trade != nil:
			price = event.Trade.Price
		case event.Quote != nil:
			price = (event.Quote.BidPrice + event.Quote.AskPrice) / 2
		}
		if price <= 0 {
			continue
		}
		series = append(series, PricePoint{At: event.At, Close: price})
	}
	sort.Slice(series, func(i, j int) bool {
		return series[i].At.Before(series[j].At)
	})
	return series
}
