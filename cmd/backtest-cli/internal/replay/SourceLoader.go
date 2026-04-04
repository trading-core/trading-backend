package replay

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/alpaca"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

func Load(ctx context.Context, input LoadInput) (LoadOutput, error) {
	strategy, err := selectStrategy(input)
	if err != nil {
		return LoadOutput{}, err
	}
	return strategy.Load(ctx, input)
}

func selectStrategy(input LoadInput) (Strategy, error) {
	source := strings.TrimSpace(strings.ToLower(input.Source))
	if source == "" {
		source = "alpaca"
	}
	switch source {
	case "alpaca":
		return alpacaCandlesStrategy{}, nil
	case "tastytrade", "tasty_trade":
		return tastyTradeHistoricalStrategy{}, nil
	default:
		return nil, fmt.Errorf("unsupported BACKTEST_DATA_SOURCE: %s (valid values: alpaca, tastytrade)", input.Source)
	}
}

type alpacaCandlesStrategy struct{}

func (alpacaCandlesStrategy) Load(ctx context.Context, input LoadInput) (LoadOutput, error) {
	prices, err := loadCandlesFromAlpaca(ctx, alpacaLoadInput{
		Symbol:    input.Symbol,
		Timeframe: input.Timeframe,
		Limit:     input.Alpaca.Limit,
		Start:     input.Start,
		End:       input.End,
		Feed:      input.Alpaca.Feed,
	})
	if err != nil {
		return LoadOutput{}, err
	}
	if len(prices) == 0 {
		return LoadOutput{}, fmt.Errorf("alpaca returned no candle rows (symbol=%s timeframe=%s start=%q end=%q feed=%s limit=%d)", input.Symbol, input.Timeframe, input.Start, input.End, input.Alpaca.Feed, input.Alpaca.Limit)
	}
	indicatorPrices := prices
	if input.WarmupBars > 0 && strings.TrimSpace(input.Start) != "" {
		warmupStart, warmupErr := computeIndicatorWarmupStart(input.Start, input.Timeframe, input.WarmupBars)
		if warmupErr == nil {
			warmupPrices, loadErr := loadCandlesFromAlpaca(ctx, alpacaLoadInput{
				Symbol:    input.Symbol,
				Timeframe: input.Timeframe,
				Limit:     input.Alpaca.Limit,
				Start:     warmupStart,
				End:       input.End,
				Feed:      input.Alpaca.Feed,
			})
			if loadErr == nil && len(warmupPrices) > len(prices) {
				indicatorPrices = warmupPrices
			}
		}
	}
	events := EventsFromCandles(input.Symbol, prices)
	return LoadOutput{Prices: prices, IndicatorPrices: indicatorPrices, Events: events}, nil
}

type tastyTradeHistoricalStrategy struct{}

func (tastyTradeHistoricalStrategy) Load(ctx context.Context, input LoadInput) (LoadOutput, error) {
	fromTime, err := parseOptionalTime(input.Start)
	if err != nil {
		return LoadOutput{}, fmt.Errorf("invalid start time: %w", err)
	}
	endTime, err := parseOptionalTime(input.End)
	if err != nil {
		return LoadOutput{}, fmt.Errorf("invalid end time: %w", err)
	}
	prices, err := loadCandlesFromTastyTrade(ctx, tastyTradeLoadInput{
		Symbol:            input.Symbol,
		BrokerType:        input.TastyTrade.BrokerType,
		CandleInterval:    input.Timeframe,
		FromTime:          fromTime,
		EndTime:           endTime,
		CollectionTimeout: input.TastyTrade.CollectionTimeout,
		MaxCandles:        input.TastyTrade.MaxCandles,
	})
	if err != nil {
		return LoadOutput{}, err
	}
	if len(prices) == 0 {
		return LoadOutput{}, fmt.Errorf("tastytrade returned no historical candle rows (symbol=%s interval=%s)", input.Symbol, input.Timeframe)
	}
	prices = filterToMarketHours(prices, input.Timeframe)
	if len(prices) == 0 {
		return LoadOutput{}, fmt.Errorf("tastytrade returned no market-hours candle rows (symbol=%s interval=%s)", input.Symbol, input.Timeframe)
	}
	indicatorPrices := prices
	if input.WarmupBars > 0 && !fromTime.IsZero() {
		warmupStart, warmupErr := computeIndicatorWarmupStart(input.Start, input.Timeframe, input.WarmupBars)
		if warmupErr == nil {
			warmupFromTime, parseErr := parseOptionalTime(warmupStart)
			if parseErr == nil {
				warmupPrices, loadErr := loadCandlesFromTastyTrade(ctx, tastyTradeLoadInput{
					Symbol:            input.Symbol,
					BrokerType:        input.TastyTrade.BrokerType,
					CandleInterval:    input.Timeframe,
					FromTime:          warmupFromTime,
					EndTime:           endTime,
					CollectionTimeout: input.TastyTrade.CollectionTimeout,
					MaxCandles:        input.TastyTrade.MaxCandles,
				})
				if loadErr == nil {
					warmupPrices = filterToMarketHours(warmupPrices, input.Timeframe)
					if len(warmupPrices) > len(prices) {
						indicatorPrices = warmupPrices
					}
				}
			}
		}
	}
	events := EventsFromCandles(input.Symbol, prices)
	return LoadOutput{Prices: prices, IndicatorPrices: indicatorPrices, Events: events}, nil
}

type alpacaLoadInput struct {
	Symbol    string
	Timeframe string
	Limit     int
	Start     string
	End       string
	Feed      string
}

func loadCandlesFromAlpaca(ctx context.Context, input alpacaLoadInput) ([]PricePoint, error) {
	if input.Symbol == "" {
		return nil, errors.New("symbol is required for alpaca source")
	}
	if input.Timeframe == "" {
		return nil, errors.New("alpaca timeframe is required")
	}
	client := alpaca.ClientFromEnv()
	barsOutput, err := client.GetStockBars(ctx, alpaca.GetStockBarsInput{
		Symbol:    input.Symbol,
		Timeframe: input.Timeframe,
		Limit:     input.Limit,
		Feed:      input.Feed,
		Start:     input.Start,
		End:       input.End,
	})
	if err != nil {
		return nil, err
	}
	points := make([]PricePoint, 0, len(barsOutput.Bars))
	for _, bar := range barsOutput.Bars {
		at, err := parseTimestamp(bar.Time)
		if err != nil {
			return nil, fmt.Errorf("invalid alpaca bar time %q: %w", bar.Time, err)
		}
		points = append(points, PricePoint{At: at, Close: bar.Close})
	}
	sort.Slice(points, func(i, j int) bool {
		return points[i].At.Before(points[j].At)
	})
	return points, nil
}

type tastyTradeLoadInput struct {
	Symbol            string
	BrokerType        string
	CandleInterval    string
	FromTime          time.Time
	EndTime           time.Time
	CollectionTimeout time.Duration
	MaxCandles        int
}

func loadCandlesFromTastyTrade(ctx context.Context, input tastyTradeLoadInput) ([]PricePoint, error) {
	if strings.TrimSpace(input.Symbol) == "" {
		return nil, errors.New("symbol is required for tastytrade source")
	}
	brokerType := strings.TrimSpace(input.BrokerType)
	if brokerType == "" {
		brokerType = "tastytrade"
	}
	credentialsByType := auth.CredentialsByTypeFromEnv()
	credentials, ok := credentialsByType[brokerType]
	if !ok {
		return nil, fmt.Errorf("tastytrade credentials not found for broker type: %s", brokerType)
	}
	apiURL, err := url.Parse(credentials.APIURL)
	if err != nil {
		return nil, fmt.Errorf("invalid tastytrade api url: %w", err)
	}
	tokenManager := auth.NewTastyTradeTokenManager(&credentials.AuthorizationServer)
	client := tastytrade.NewHTTPClient(tastytrade.NewHTTPClientInput{
		APIURL:         apiURL,
		GetAccessToken: tokenManager.GetAccessToken,
	})
	adapter := broker.NewTastyTradeMarketDataAdapter(broker.NewTastyTradeMarketDataAdapterInput{Client: client})
	timeout := input.CollectionTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	maxCandles := input.MaxCandles
	if maxCandles <= 0 {
		maxCandles = 2500
	}
	bucketDuration, err := candleIntervalToDuration(input.CandleInterval)
	if err != nil {
		return nil, err
	}
	streamCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	iterator := adapter.GetHistoricalData(streamCtx, broker.GetHistoricaDataInput{
		Symbol:         input.Symbol,
		CandleInterval: input.CandleInterval,
		FromTime:       input.FromTime,
	})
	byTs := make(map[int64]PricePoint, maxCandles)
	for iterator.Next() {
		message := iterator.Item()
		if message == nil || message.Candle == nil || message.ReceivedAt.IsZero() {
			continue
		}
		if message.Candle.Close <= 0 || math.IsNaN(message.Candle.Close) || math.IsInf(message.Candle.Close, 0) {
			continue
		}
		if !input.FromTime.IsZero() && message.ReceivedAt.Before(input.FromTime) {
			continue
		}
		if !input.EndTime.IsZero() && message.ReceivedAt.After(input.EndTime) {
			continue
		}
		bucketAt := bucketTime(message.ReceivedAt, bucketDuration)
		ts := bucketAt.UnixMilli()
		byTs[ts] = PricePoint{At: bucketAt, Close: message.Candle.Close}
		if len(byTs) >= maxCandles {
			break
		}
	}
	if err := iterator.Err(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	if len(byTs) == 0 {
		return nil, nil
	}
	points := make([]PricePoint, 0, len(byTs))
	for _, point := range byTs {
		points = append(points, point)
	}
	sort.Slice(points, func(i, j int) bool {
		return points[i].At.Before(points[j].At)
	})
	return points, nil
}

func parseTimestamp(value string) (time.Time, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	layouts := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02 15:04"}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, clean); err == nil {
			return ts, nil
		}
	}
	if unixSeconds, err := strconv.ParseInt(clean, 10, 64); err == nil {
		return time.Unix(unixSeconds, 0).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", clean)
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

func candleIntervalToDuration(interval string) (time.Duration, error) {
	clean := strings.TrimSpace(strings.ToLower(interval))
	if clean == "" {
		return time.Minute, nil
	}
	if strings.HasSuffix(clean, "min") {
		clean = strings.TrimSuffix(clean, "min") + "m"
	}
	if strings.HasSuffix(clean, "hour") {
		clean = strings.TrimSuffix(clean, "hour") + "h"
	}
	if strings.HasSuffix(clean, "day") {
		clean = strings.TrimSuffix(clean, "day") + "d"
	}
	if strings.HasSuffix(clean, "week") {
		clean = strings.TrimSuffix(clean, "week") + "w"
	}
	if len(clean) < 2 {
		return 0, fmt.Errorf("unsupported tastytrade candle interval: %s", interval)
	}
	unit := clean[len(clean)-1]
	n, err := strconv.Atoi(clean[:len(clean)-1])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("unsupported tastytrade candle interval: %s", interval)
	}
	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported tastytrade candle interval: %s", interval)
	}
}

func bucketTime(t time.Time, interval time.Duration) time.Time {
	if interval <= 0 {
		return t.UTC()
	}
	return t.UTC().Truncate(interval)
}

var usEastern = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	return loc
}()

func filterToMarketHours(prices []PricePoint, timeframe string) []PricePoint {
	// For daily and weekly timeframes, don't filter to market hours (they close outside 9:30-16:00 window).
	// Only filter intraday (1Min, 5Min, etc.) to 9:30 AM - 4:00 PM.
	if timeframe == "1Day" || timeframe == "1Week" {
		return prices
	}
	out := make([]PricePoint, 0, len(prices))
	for _, p := range prices {
		h, m, _ := p.At.In(usEastern).Clock()
		mins := h*60 + m
		if mins >= 9*60+30 && mins <= 16*60 {
			out = append(out, p)
		}
	}
	return out
}
