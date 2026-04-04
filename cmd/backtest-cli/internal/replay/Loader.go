package replay

import (
	"sort"
)

func EventsFromCandles(symbol string, candles []PricePoint) []Event {
	events := make([]Event, 0, len(candles))
	for _, c := range candles {
		events = append(events, Event{
			Type:   EventTypeTrade,
			At:     c.At,
			Symbol: symbol,
			Trade:  &Trade{Price: c.Close},
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
