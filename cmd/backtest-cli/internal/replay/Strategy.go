package replay

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Strategy interface {
	Load(ctx context.Context, input LoadInput) (*LoadOutput, error)
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

func (input LoadInput) SelectStrategy() (strategy Strategy, err error) {
	source := strings.TrimSpace(strings.ToLower(input.Source))
	if source == "" {
		source = "alpaca"
	}
	switch source {
	case "alpaca":
		strategy = new(alpacaCandleStrategy)
		return
	case "tastytrade":
		strategy = new(tastyTradeHistoricalStrategy)
		return
	default:
		err = fmt.Errorf("unsupported BACKTEST_DATA_SOURCE: %s (valid values: alpaca, tastytrade)", input.Source)
		return
	}
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
