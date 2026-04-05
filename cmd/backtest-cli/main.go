package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/backtest"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/backtestconfig"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/chart"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/indicator"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

func main() {
	ctx := context.Background()
	cfg := backtestconfig.LoadFromEnv()
	replayInput := cfg.ReplayInput()
	strategy, err := replayInput.SelectStrategy()
	fatal.OnError(err)
	loaded, err := strategy.Load(ctx, replayInput)
	fatal.OnError(err)
	if cfg.Tune {
		RunTune(cfg, loaded)
		return
	}
	outputDir := cfg.OutputDir()
	err = os.MkdirAll(outputDir, 0o755)
	fatal.OnError(err)
	RunBacktestAndPlot(cfg, loaded, outputDir)
}

func RunTune(cfg backtestconfig.Config, loaded *replay.LoadOutput) {
	result := backtest.Run(cfg, loaded.Prices, loaded.Events)
	out := struct {
		TotalReturn float64 `json:"total_return"`
		Sharpe      float64 `json:"sharpe"`
		Trades      int     `json:"trades"`
		WinRate     float64 `json:"win_rate"`
	}{
		TotalReturn: result.TotalReturn,
		Sharpe:      result.SharpeRatio,
		Trades:      result.TradeCount,
		WinRate:     result.WinRate,
	}
	b, err := json.Marshal(out)
	fatal.OnError(err)
	fmt.Println(string(b))
}

func RunBacktestAndPlot(cfg backtestconfig.Config, loaded *replay.LoadOutput, outputDir string) {
	result := backtest.Run(cfg, loaded.Prices, loaded.Events)
	plotStart := result.Prices[0].At
	plotEnd := result.Prices[len(result.Prices)-1].At
	rsiSeries := indicator.ComputeRSI(loaded.IndicatorPrices, cfg.Indicators.RSIPeriod)
	macdSeries, macdSignalSeries := indicator.ComputeMACD(loaded.IndicatorPrices, cfg.Indicators.MACDFastPeriod, cfg.Indicators.MACDSlowPeriod, cfg.Indicators.MACDSignalPeriod)
	bollUpperSeries, bollMiddleSeries, bollLowerSeries := indicator.ComputeBollingerBands(loaded.IndicatorPrices, cfg.Indicators.BollingerPeriod, cfg.Indicators.BollingerStdDev)
	smaSeries := indicator.ComputeSMA(loaded.IndicatorPrices, cfg.Indicators.SMAPeriod)
	tz := tradingstrategy.USMarketLocation
	rsiForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(rsiSeries, plotStart, plotEnd), tz, cfg.TradingParameters.Timeframe)
	macdForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(macdSeries, plotStart, plotEnd), tz, cfg.TradingParameters.Timeframe)
	macdSignalForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(macdSignalSeries, plotStart, plotEnd), tz, cfg.TradingParameters.Timeframe)
	bollUpperForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(bollUpperSeries, plotStart, plotEnd), tz, cfg.TradingParameters.Timeframe)
	bollMiddleForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(bollMiddleSeries, plotStart, plotEnd), tz, cfg.TradingParameters.Timeframe)
	bollLowerForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(bollLowerSeries, plotStart, plotEnd), tz, cfg.TradingParameters.Timeframe)
	smaForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(smaSeries, plotStart, plotEnd), tz, cfg.TradingParameters.Timeframe)
	outputCombinedPNG := fmt.Sprintf("%s/backtest-with-indicators.png", outputDir)
	err := chart.RenderCombined(chart.RenderCombinedInput{
		Symbol:      result.Symbol,
		TotalReturn: result.TotalReturn,
		Prices:      chartPrices(result.Prices),
		Decisions:   chartDecisions(result.Decisions),
		BollUpper:   chartIndicatorPoints(bollUpperForPlot),
		BollMiddle:  chartIndicatorPoints(bollMiddleForPlot),
		BollLower:   chartIndicatorPoints(bollLowerForPlot),
		SMA:         chartIndicatorPoints(smaForPlot),
		SMAPeriod:   cfg.Indicators.SMAPeriod,
		RSI:         chartIndicatorPoints(rsiForPlot),
		MACD:        chartIndicatorPoints(macdForPlot),
		MACDSignal:  chartIndicatorPoints(macdSignalForPlot),
		RSIPeriod:   cfg.Indicators.RSIPeriod,
		MACDFast:    cfg.Indicators.MACDFastPeriod,
		MACDSlow:    cfg.Indicators.MACDSlowPeriod,
		MACDSignalN: cfg.Indicators.MACDSignalPeriod,
		Timezone:    tz,
	}, outputCombinedPNG)
	fatal.OnError(err)
	outputPNG := fmt.Sprintf("%s/backtest.png", outputDir)
	err = chart.Render(chart.RenderInput{
		Symbol:      result.Symbol,
		TotalReturn: result.TotalReturn,
		Prices:      chartPrices(result.Prices),
		Decisions:   chartDecisions(result.Decisions),
		BollUpper:   chartIndicatorPoints(bollUpperForPlot),
		BollMiddle:  chartIndicatorPoints(bollMiddleForPlot),
		BollLower:   chartIndicatorPoints(bollLowerForPlot),
		SMA:         chartIndicatorPoints(smaForPlot),
		SMAPeriod:   cfg.Indicators.SMAPeriod,
		Timezone:    tz,
	}, outputPNG)
	fatal.OnError(err)
	outputIndicatorsPNG := fmt.Sprintf("%s/indicators.png", outputDir)
	err = chart.RenderIndicators(chart.RenderIndicatorsInput{
		Symbol:      result.Symbol,
		Timeline:    chartTimes(result.Prices),
		RSI:         chartIndicatorPoints(rsiForPlot),
		MACD:        chartIndicatorPoints(macdForPlot),
		MACDSignal:  chartIndicatorPoints(macdSignalForPlot),
		RSIPeriod:   cfg.Indicators.RSIPeriod,
		MACDFast:    cfg.Indicators.MACDFastPeriod,
		MACDSlow:    cfg.Indicators.MACDSlowPeriod,
		MACDSignalN: cfg.Indicators.MACDSignalPeriod,
		Timezone:    tz,
	}, outputIndicatorsPNG)
	fatal.OnError(err)

	fmt.Printf("Backtest complete for %s\n", result.Symbol)
	fmt.Printf("Rows: %d\n", len(result.Prices))
	fmt.Printf("Decisions: %d\n", len(result.Decisions))
	fmt.Printf("Starting cash: %.2f\n", result.StartingCash)
	fmt.Printf("Ending cash: %.2f\n", result.EndingCash)
	fmt.Printf("Ending value: %.2f\n", result.EndingValue)
	fmt.Printf("Total return: %.2f%%\n", result.TotalReturn*100)
	fmt.Printf("Combined image: %s\n", outputCombinedPNG)
	fmt.Printf("Output image: %s\n", outputPNG)
	fmt.Printf("Indicators image: %s\n", outputIndicatorsPNG)
}

func filterIndicatorSeriesToRange(points []indicator.Point, start time.Time, end time.Time) []indicator.Point {
	out := make([]indicator.Point, 0, len(points))
	for _, p := range points {
		if p.At.Before(start) || p.At.After(end) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func filterIndicatorToMarketHours(points []indicator.Point, tz *time.Location, timeframe string) []indicator.Point {
	// For daily and weekly timeframes, don't filter to market hours (they need end-of-day/week closes).
	// Only filter intraday (1Min, 5Min, etc.) to 9:30 AM - 4:00 PM.
	if timeframe == "1Day" || timeframe == "1Week" {
		return points
	}
	out := make([]indicator.Point, 0, len(points))
	for _, p := range points {
		local := p.At.In(tz)
		h, m, _ := local.Clock()
		mins := h*60 + m
		if mins >= 9*60+30 && mins <= 16*60 {
			out = append(out, p)
		}
	}
	return out
}

func chartTimes(prices []replay.PricePoint) []time.Time {
	out := make([]time.Time, len(prices))
	for i, p := range prices {
		out[i] = p.At
	}
	return out
}

func chartIndicatorPoints(points []indicator.Point) []chart.IndicatorPoint {
	out := make([]chart.IndicatorPoint, len(points))
	for i, p := range points {
		out[i] = chart.IndicatorPoint{At: p.At, Value: p.Value}
	}
	return out
}

func chartPrices(prices []replay.PricePoint) []chart.PricePoint {
	out := make([]chart.PricePoint, len(prices))
	for i, p := range prices {
		out[i] = chart.PricePoint{At: p.At, Close: p.Close}
	}
	return out
}

func chartDecisions(decisions []backtest.DecisionPoint) []chart.DecisionMarker {
	out := make([]chart.DecisionMarker, len(decisions))
	for i, d := range decisions {
		out[i] = chart.DecisionMarker{
			At:    d.At,
			Price: d.Price,
			IsBuy: d.Action == tradingstrategy.ActionBuy,
		}
	}
	return out
}
