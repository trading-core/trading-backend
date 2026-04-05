package replay

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Strategy interface {
	Load(ctx context.Context, input LoadInput) (*LoadOutput, error)
}

type LoadInput struct {
	Source       string
	Symbol       string
	Timeframe    string // canonical candle interval, e.g. "1Min", "1Hour", "1Day"
	Start        string // RFC 3339 start time (inclusive)
	End          string // RFC 3339 end time (inclusive), may be empty
	WarmupBars   int
	CacheEnabled bool
	CacheDir     string
	Alpaca       AlpacaInput
	TastyTrade   TastyTradeInput
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

func (input LoadInput) SelectStrategy() (strategy Strategy, err error) {
	source := strings.TrimSpace(strings.ToLower(input.Source))
	if source == "" {
		source = "alpaca"
	}
	switch source {
	case "alpaca":
		strategy = new(alpacaCandleStrategy)
	case "tastytrade":
		strategy = new(tastyTradeHistoricalStrategy)
	default:
		err = fmt.Errorf("unsupported BACKTEST_DATA_SOURCE: %s (valid values: alpaca, tastytrade)", input.Source)
		return
	}
	if input.CacheEnabled {
		strategy = &cacheDecorator{base: strategy}
	}
	return
}

type LoadOutput struct {
	Prices          []PricePoint
	IndicatorPrices []PricePoint
	Events          []Event
}

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
	for _, candle := range candles {
		events = append(events, Event{
			Type:   EventTypeTrade,
			At:     candle.At,
			Symbol: symbol,
			Trade: &Trade{
				Price: candle.Close,
			},
		})
	}
	return events
}

func CandlesFromEvents(events []Event) []PricePoint {
	candles := make([]PricePoint, 0, len(events))
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
		candles = append(candles, PricePoint{At: event.At, Close: price})
	}
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].At.Before(candles[j].At)
	})
	return candles
}

func parseTimestamp(value string) (timestamp time.Time, err error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		err = errors.New("empty timestamp")
		return
	}
	// Try multiple layouts for flexibility (RFC3339, RFC3339Nano, common datetime formats).
	layouts := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02 15:04"}
	for _, layout := range layouts {
		timestamp, err = time.Parse(layout, clean)
		if err != nil {
			continue
		}
		return
	}
	unixSeconds, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		err = fmt.Errorf("unsupported time format: %s", clean)
		return
	}
	timestamp = time.Unix(unixSeconds, 0).UTC()
	return
}

func parseOptionalTime(value string) (time.Time, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, clean)
}

func computeIndicatorWarmupStart(startRFC3339 string, timeframe string, warmupBars int) (string, error) {
	if warmupBars <= 0 {
		return startRFC3339, nil
	}
	startAt, err := parseTimestamp(startRFC3339)
	if err != nil {
		return "", fmt.Errorf("invalid BACKTEST_START: %w", err)
	}
	barSize, err := timeframeToDuration(timeframe)
	if err != nil {
		return "", err
	}
	warmupDuration := time.Duration(warmupBars) * barSize
	return startAt.Add(-warmupDuration).Format(time.RFC3339), nil
}

func timeframeToDuration(timeframe string) (time.Duration, error) {
	clean := strings.TrimSpace(strings.ToLower(timeframe))
	switch clean {
	case "1min", "1m":
		return time.Minute, nil
	case "5min", "5m":
		return 5 * time.Minute, nil
	case "15min", "15m":
		return 15 * time.Minute, nil
	case "1hour", "1h":
		return time.Hour, nil
	case "1day", "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported timeframe for warmup: %s", timeframe)
	}
}
