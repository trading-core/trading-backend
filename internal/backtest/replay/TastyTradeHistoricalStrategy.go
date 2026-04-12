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
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type tastyTradeHistoricalStrategy struct{}

func (strategy *tastyTradeHistoricalStrategy) Load(ctx context.Context, input LoadInput) (output *LoadOutput, err error) {
	fromTime, err := parseOptionalTime(input.Start)
	if err != nil {
		err = fmt.Errorf("invalid start time: %w", err)
		return
	}
	endTime, err := parseOptionalTime(input.End)
	if err != nil {
		err = fmt.Errorf("invalid end time: %w", err)
		return
	}
	prices, err := strategy.loadCandlesFromTastyTrade(ctx, tastyTradeLoadInput{
		Symbol:            input.Symbol,
		BrokerType:        input.TastyTrade.BrokerType,
		CandleInterval:    input.Timeframe,
		FromTime:          fromTime,
		EndTime:           endTime,
		CollectionTimeout: input.TastyTrade.CollectionTimeout,
		MaxCandles:        input.TastyTrade.MaxCandles,
	})
	if err != nil {
		err = fmt.Errorf("failed to load candles from tastytrade: %w", err)
		return
	}
	if len(prices) == 0 {
		err = fmt.Errorf("tastytrade returned no historical candle rows (symbol=%s interval=%s)", input.Symbol, input.Timeframe)
		return
	}
	prices = filterToMarketHours(prices, input.Timeframe)
	if len(prices) == 0 {
		err = fmt.Errorf("tastytrade returned no market-hours candle rows (symbol=%s interval=%s)", input.Symbol, input.Timeframe)
		return
	}
	indicatorPrices, err := strategy.getIndicatorPrices(ctx, input, fromTime, endTime, prices)
	if err != nil {
		err = fmt.Errorf("failed to get indicator prices: %w", err)
		return
	}
	output = &LoadOutput{
		Prices:          prices,
		IndicatorPrices: indicatorPrices,
		Events:          EventsFromCandles(input.Symbol, prices),
	}
	return
}

func (strategy *tastyTradeHistoricalStrategy) getIndicatorPrices(ctx context.Context, input LoadInput, fromTime time.Time, endTime time.Time, prices []PricePoint) (indicatorPrices []PricePoint, err error) {
	hasWarmupBarsAndStart := input.WarmupBars > 0 && !fromTime.IsZero()
	if !hasWarmupBarsAndStart {
		indicatorPrices = prices
		return
	}
	warmupStart, err := computeIndicatorWarmupStart(input.Start, input.Timeframe, input.WarmupBars)
	if err != nil {
		return
	}
	warmupFromTime, err := parseOptionalTime(warmupStart)
	if err != nil {
		return
	}
	indicatorPrices, err = strategy.loadCandlesFromTastyTrade(ctx, tastyTradeLoadInput{
		Symbol:            input.Symbol,
		BrokerType:        input.TastyTrade.BrokerType,
		CandleInterval:    input.Timeframe,
		FromTime:          warmupFromTime,
		EndTime:           endTime,
		CollectionTimeout: input.TastyTrade.CollectionTimeout,
		MaxCandles:        input.TastyTrade.MaxCandles,
	})
	if err != nil {
		return
	}
	indicatorPrices = filterToMarketHours(indicatorPrices, input.Timeframe)
	return
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

func (strategy *tastyTradeHistoricalStrategy) loadCandlesFromTastyTrade(ctx context.Context, input tastyTradeLoadInput) ([]PricePoint, error) {
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
	bucketDuration, err := strategy.candleIntervalToDuration(input.CandleInterval)
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

func (strategy *tastyTradeHistoricalStrategy) candleIntervalToDuration(interval string) (time.Duration, error) {
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
