package replay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kduong/trading-backend/internal/broker"
)

func LoadEventsFromFile(path string, symbol string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []Event
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var message broker.MarketDataMessage
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			return nil, fmt.Errorf("invalid replay JSON at line %d: %w", lineNumber, err)
		}
		if symbol != "" && message.Symbol != symbol {
			continue
		}
		if message.ReceivedAt.IsZero() {
			return nil, fmt.Errorf("missing received_at at line %d", lineNumber)
		}
		switch message.Type {
		case broker.MarketDataTypeTrade:
			if message.Trade == nil {
				continue
			}
			events = append(events, Event{
				Type:      EventTypeTrade,
				At:        message.ReceivedAt,
				Symbol:    message.Symbol,
				Trade:     &Trade{Price: message.Trade.Price},
				DayVolume: cloneFloat64Ptr(message.Trade.DayVolume),
				Size:      cloneFloat64Ptr(message.Trade.Size),
			})
		case broker.MarketDataTypeQuote:
			if message.Quote == nil {
				continue
			}
			events = append(events, Event{
				Type:   EventTypeQuote,
				At:     message.ReceivedAt,
				Symbol: message.Symbol,
				Quote: &Quote{
					BidPrice: message.Quote.BidPrice,
					AskPrice: message.Quote.AskPrice,
					BidSize:  message.Quote.BidSize,
					AskSize:  message.Quote.AskSize,
				},
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].At.Before(events[j].At)
	})
	return events, nil
}

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
