package replay

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kduong/trading-backend/internal/broker/alpaca"
)

type alpacaCandleStrategy struct{}

func (strategy *alpacaCandleStrategy) Load(ctx context.Context, input LoadInput) (output *LoadOutput, err error) {
	prices, err := strategy.loadCandlesFromAlpaca(ctx, alpacaLoadInput{
		Symbol:    input.Symbol,
		Timeframe: input.Timeframe,
		Limit:     input.Alpaca.Limit,
		Start:     input.Start,
		End:       input.End,
		Feed:      input.Alpaca.Feed,
	})
	if err != nil {
		return
	}
	if len(prices) == 0 {
		err = fmt.Errorf("alpaca returned no candle rows (symbol=%s timeframe=%s start=%q end=%q feed=%s limit=%d)", input.Symbol, input.Timeframe, input.Start, input.End, input.Alpaca.Feed, input.Alpaca.Limit)
		return
	}
	indicatorPrices, err := strategy.getIndicatorPrices(ctx, input, prices)
	if err != nil {
		return
	}
	output = &LoadOutput{
		Prices:          prices,
		IndicatorPrices: indicatorPrices,
		Events:          EventsFromCandles(input.Symbol, prices),
	}
	return
}

func (strategy *alpacaCandleStrategy) getIndicatorPrices(ctx context.Context, input LoadInput, prices []PricePoint) (indicatorPrices []PricePoint, err error) {
	hasWarmupBarsAndStart := input.WarmupBars > 0 && strings.TrimSpace(input.Start) != ""
	if !hasWarmupBarsAndStart {
		indicatorPrices = prices
		return
	}
	warmupStart, err := computeIndicatorWarmupStart(input.Start, input.Timeframe, input.WarmupBars)
	if err != nil {
		return
	}
	return strategy.loadCandlesFromAlpaca(ctx, alpacaLoadInput{
		Symbol:    input.Symbol,
		Timeframe: input.Timeframe,
		Limit:     input.Alpaca.Limit,
		Start:     warmupStart,
		End:       input.End,
		Feed:      input.Alpaca.Feed,
	})
}

type alpacaLoadInput struct {
	Symbol    string
	Timeframe string
	Limit     int
	Start     string
	End       string
	Feed      string
}

func (strategy *alpacaCandleStrategy) loadCandlesFromAlpaca(ctx context.Context, input alpacaLoadInput) (points []PricePoint, err error) {
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
		return
	}
	points = make([]PricePoint, 0, len(barsOutput.Bars))
	for _, bar := range barsOutput.Bars {
		var at time.Time
		at, err = parseTimestamp(bar.Time)
		if err != nil {
			err = fmt.Errorf("invalid alpaca bar time %q: %w", bar.Time, err)
			return
		}
		points = append(points, PricePoint{
			At:    at,
			Close: bar.Close,
		})
	}
	sort.Slice(points, func(i, j int) bool {
		return points[i].At.Before(points[j].At)
	})
	return
}
