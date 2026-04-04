package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/backtestconfig"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/chart"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

type decisionPoint struct {
	At       time.Time
	Price    float64
	Action   tradingstrategy.Action
	Quantity float64
	Reason   string
}

type pendingOrder struct {
	Action   tradingstrategy.Action
	Quantity float64
	Reason   string
	FillAt   time.Time
}

func main() {
	ctx := context.Background()
	cfg, err := backtestconfig.LoadFromEnv()
	fatal.OnError(err)
	loaded, err := replay.Load(ctx, cfg.ReplayInput())
	fatal.OnError(err)
	outputDir := cfg.OutputDir()
	err = os.MkdirAll(outputDir, 0o755)
	fatal.OnError(err)
	if cfg.Sweep {
		runSweep(cfg, loaded.Prices, loaded.Events, outputDir)
		return
	}
	backTestResult := runBacktest(cfg, loaded.Prices, loaded.Events)
	plotStart := backTestResult.Prices[0].At
	plotEnd := backTestResult.Prices[len(backTestResult.Prices)-1].At
	ind := cfg.Indicators
	rsiSeries := computeRSI(loaded.IndicatorPrices, ind.RSIPeriod)
	macdSeries, macdSignalSeries := computeMACD(loaded.IndicatorPrices, ind.MACDFastPeriod, ind.MACDSlowPeriod, ind.MACDSignalPeriod)
	bollUpperSeries, bollMiddleSeries, bollLowerSeries := computeBollingerBands(loaded.IndicatorPrices, ind.BollingerPeriod, ind.BollingerStdDev)
	tz := tradingstrategy.USMarketLocation
	rsiForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(rsiSeries, plotStart, plotEnd), tz, cfg.Timeframe)
	macdForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(macdSeries, plotStart, plotEnd), tz, cfg.Timeframe)
	macdSignalForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(macdSignalSeries, plotStart, plotEnd), tz, cfg.Timeframe)
	bollUpperForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(bollUpperSeries, plotStart, plotEnd), tz, cfg.Timeframe)
	bollMiddleForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(bollMiddleSeries, plotStart, plotEnd), tz, cfg.Timeframe)
	bollLowerForPlot := filterIndicatorToMarketHours(filterIndicatorSeriesToRange(bollLowerSeries, plotStart, plotEnd), tz, cfg.Timeframe)
	outputCombinedPNG := fmt.Sprintf("%s/backtest-with-indicators.png", outputDir)
	err = chart.RenderCombined(chart.RenderCombinedInput{
		Symbol:      backTestResult.Symbol,
		Strategy:    backTestResult.Strategy,
		TotalReturn: backTestResult.TotalReturn,
		Prices:      chartPrices(backTestResult.Prices),
		Decisions:   chartDecisions(backTestResult.Decisions),
		BollUpper:   chartIndicatorPoints(bollUpperForPlot),
		BollMiddle:  chartIndicatorPoints(bollMiddleForPlot),
		BollLower:   chartIndicatorPoints(bollLowerForPlot),
		RSI:         chartIndicatorPoints(rsiForPlot),
		MACD:        chartIndicatorPoints(macdForPlot),
		MACDSignal:  chartIndicatorPoints(macdSignalForPlot),
		RSIPeriod:   ind.RSIPeriod,
		MACDFast:    ind.MACDFastPeriod,
		MACDSlow:    ind.MACDSlowPeriod,
		MACDSignalN: ind.MACDSignalPeriod,
		Timezone:    tz,
	}, outputCombinedPNG)
	fatal.OnError(err)
	outputPNG := fmt.Sprintf("%s/backtest.png", outputDir)
	err = chart.Render(chart.RenderInput{
		Symbol:      backTestResult.Symbol,
		Strategy:    backTestResult.Strategy,
		TotalReturn: backTestResult.TotalReturn,
		Prices:      chartPrices(backTestResult.Prices),
		Decisions:   chartDecisions(backTestResult.Decisions),
		BollUpper:   chartIndicatorPoints(bollUpperForPlot),
		BollMiddle:  chartIndicatorPoints(bollMiddleForPlot),
		BollLower:   chartIndicatorPoints(bollLowerForPlot),
		Timezone:    tz,
	}, outputPNG)
	fatal.OnError(err)
	outputIndicatorsPNG := fmt.Sprintf("%s/indicators.png", outputDir)
	err = chart.RenderIndicators(chart.RenderIndicatorsInput{
		Symbol:      backTestResult.Symbol,
		Strategy:    backTestResult.Strategy,
		Timeline:    chartTimes(backTestResult.Prices),
		RSI:         chartIndicatorPoints(rsiForPlot),
		MACD:        chartIndicatorPoints(macdForPlot),
		MACDSignal:  chartIndicatorPoints(macdSignalForPlot),
		RSIPeriod:   ind.RSIPeriod,
		MACDFast:    ind.MACDFastPeriod,
		MACDSlow:    ind.MACDSlowPeriod,
		MACDSignalN: ind.MACDSignalPeriod,
		Timezone:    tz,
	}, outputIndicatorsPNG)
	fatal.OnError(err)

	fmt.Printf("Backtest complete for %s (%s)\n", backTestResult.Symbol, backTestResult.Strategy)
	fmt.Printf("Rows: %d\n", len(backTestResult.Prices))
	fmt.Printf("Decisions: %d\n", len(backTestResult.Decisions))
	fmt.Printf("Starting cash: %.2f\n", backTestResult.StartingCash)
	fmt.Printf("Ending cash: %.2f\n", backTestResult.EndingCash)
	fmt.Printf("Ending value: %.2f\n", backTestResult.EndingValue)
	fmt.Printf("Total return: %.2f%%\n", backTestResult.TotalReturn*100)
	fmt.Printf("Combined image: %s\n", outputCombinedPNG)
	fmt.Printf("Output image: %s\n", outputPNG)
	fmt.Printf("Indicators image: %s\n", outputIndicatorsPNG)
}

func runSweep(cfg backtestconfig.Config, prices []replay.PricePoint, events []replay.Event, outputDir string) {
	// Practical TP ladder from 1.5% up to 20%.
	takeProfitValues := []float64{0.015, 0.02, 0.03, 0.05, 0.075, 0.10, 0.125, 0.15, 0.175, 0.20}
	positionValues := []float64{0.05, 0.10, 0.15, 0.20, 0.25, 0.30}
	sessionStartValues := []int{10, 11}
	sessionEndValues := []int{14, 15, 16}

	// Split data into per-day segments so each param combo is tested across
	// multiple sessions to avoid single-day overfitting.
	days := splitByTradingDay(events, prices)
	fatal.Unlessf(len(days) > 0, "no trading days found in data")
	fmt.Fprintf(os.Stderr, "sweep: %d trading day(s) detected\n", len(days))

	// Window sizes for multi-day sessions. Window=1 is single-day (original
	// behaviour). Larger windows carry position/cash across consecutive days.
	var windowSizes []int
	for w := 1; w <= len(days); w++ {
		windowSizes = append(windowSizes, w)
	}

	type sweepResult struct {
		TakeProfit   float64
		Position     float64
		Start        int
		End          int
		WindowDays   int
		AvgReturn    float64
		WinWindows   int
		TotalWindows int
		TotalTrades  int
	}

	var results []sweepResult

	combos := len(takeProfitValues) * len(positionValues) * len(sessionStartValues) * len(sessionEndValues) * len(windowSizes)
	run := 0
	for _, tp := range takeProfitValues {
		for _, pos := range positionValues {
			for _, ss := range sessionStartValues {
				for _, se := range sessionEndValues {
					for _, ws := range windowSizes {
						run++
						sweepCfg := cfg
						sweepCfg.Scalping.TakeProfitPct = tp
						sweepCfg.Scalping.MaxPositionFraction = pos
						sweepCfg.Scalping.SessionStart = ss
						sweepCfg.Scalping.SessionEnd = se
						var totalReturn float64
						var winWindows, totalTrades, windows int
						for i := 0; i+ws <= len(days); i++ {
							windowEvents, windowPrices := mergeWindow(days[i : i+ws])
							res := runBacktest(sweepCfg, windowPrices, windowEvents)
							totalReturn += res.TotalReturn
							totalTrades += len(res.Decisions)
							if res.TotalReturn > 0 {
								winWindows++
							}
							windows++
						}
						if windows > 0 {
							results = append(results, sweepResult{
								TakeProfit:   tp,
								Position:     pos,
								Start:        ss,
								End:          se,
								WindowDays:   ws,
								AvgReturn:    totalReturn / float64(windows),
								WinWindows:   winWindows,
								TotalWindows: windows,
								TotalTrades:  totalTrades,
							})
						}
						if run%50 == 0 || run == combos {
							fmt.Fprintf(os.Stderr, "sweep: %d/%d combos\n", run, combos)
						}
					}
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].AvgReturn > results[j].AvgReturn
	})

	csvPath := fmt.Sprintf("%s/sweep-results.csv", outputDir)
	f, err := os.Create(csvPath)
	if err != nil {
		fatal.OnError(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"TakeProfit%", "Position%", "SessionStartHour(ET)", "SessionEndHour(ET)", "WindowDays", "AvgReturn%", "WinRate%", "Windows", "Trades"})
	for _, r := range results {
		winRate := float64(r.WinWindows) / float64(r.TotalWindows) * 100
		w.Write([]string{
			fmt.Sprintf("%.2f", r.TakeProfit*100),
			fmt.Sprintf("%.1f", r.Position*100),
			strconv.Itoa(r.Start),
			strconv.Itoa(r.End),
			strconv.Itoa(r.WindowDays),
			fmt.Sprintf("%.4f", r.AvgReturn*100),
			fmt.Sprintf("%.1f", winRate),
			strconv.Itoa(r.TotalWindows),
			strconv.Itoa(r.TotalTrades),
		})
	}
	w.Flush()
	fatal.OnError(w.Error())

	fmt.Fprintf(os.Stderr, "sweep results written to %s (%d rows)\n", csvPath, len(results))
}

// mergeWindow concatenates events and prices from consecutive trading days into
// a single contiguous slice for multi-day backtest sessions.
func mergeWindow(days []tradingDay) ([]replay.Event, []replay.PricePoint) {
	var events []replay.Event
	var prices []replay.PricePoint
	for _, d := range days {
		events = append(events, d.events...)
		prices = append(prices, d.prices...)
	}
	return events, prices
}

// tradingDay holds one day's worth of events and the corresponding price series.
type tradingDay struct {
	date   string // "2006-01-02"
	events []replay.Event
	prices []replay.PricePoint
}

// splitByTradingDay partitions events and prices into per-calendar-day segments
// using the US Eastern timezone so that a single market session stays together.
func splitByTradingDay(events []replay.Event, prices []replay.PricePoint) []tradingDay {
	dayEvents := make(map[string][]replay.Event)
	for _, e := range events {
		d := e.At.In(tradingstrategy.USMarketLocation).Format("2006-01-02")
		dayEvents[d] = append(dayEvents[d], e)
	}
	dayPrices := make(map[string][]replay.PricePoint)
	for _, p := range prices {
		d := p.At.In(tradingstrategy.USMarketLocation).Format("2006-01-02")
		dayPrices[d] = append(dayPrices[d], p)
	}

	var dates []string
	for d := range dayEvents {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	var days []tradingDay
	for _, d := range dates {
		p := dayPrices[d]
		if len(p) == 0 {
			p = replay.PriceSeries(dayEvents[d])
		}
		if len(p) == 0 {
			continue
		}
		days = append(days, tradingDay{date: d, events: dayEvents[d], prices: p})
	}
	return days
}

type result struct {
	Symbol        string
	Strategy      string
	StartingCash  float64
	EndingCash    float64
	EndingValue   float64
	TotalReturn   float64
	Prices        []replay.PricePoint
	Decisions     []decisionPoint
	FinalPosition float64
}

func runBacktest(cfg backtestconfig.Config, prices []replay.PricePoint, events []replay.Event) result {
	strategy := tradingstrategy.NewWithParams(cfg.Strategy, cfg.Scalping)
	ind := cfg.Indicators
	rsiSeries := computeRSI(prices, ind.RSIPeriod)
	macdSeries, macdSignalSeries := computeMACD(prices, ind.MACDFastPeriod, ind.MACDSlowPeriod, ind.MACDSignalPeriod)
	bollUpperSeries, bollMiddleSeries, bollLowerSeries := computeBollingerBands(prices, ind.BollingerPeriod, ind.BollingerStdDev)
	rsiByTs := make(map[int64]float64, len(rsiSeries))
	for _, p := range rsiSeries {
		rsiByTs[p.At.Unix()] = p.Value
	}
	macdByTs := make(map[int64]float64, len(macdSeries))
	for _, p := range macdSeries {
		macdByTs[p.At.Unix()] = p.Value
	}
	macdSignalByTs := make(map[int64]float64, len(macdSignalSeries))
	for _, p := range macdSignalSeries {
		macdSignalByTs[p.At.Unix()] = p.Value
	}
	bollUpperByTs := make(map[int64]float64, len(bollUpperSeries))
	for _, p := range bollUpperSeries {
		bollUpperByTs[p.At.Unix()] = p.Value
	}
	bollMiddleByTs := make(map[int64]float64, len(bollMiddleSeries))
	for _, p := range bollMiddleSeries {
		bollMiddleByTs[p.At.Unix()] = p.Value
	}
	bollLowerByTs := make(map[int64]float64, len(bollLowerSeries))
	for _, p := range bollLowerSeries {
		bollLowerByTs[p.At.Unix()] = p.Value
	}
	account := tradingstrategy.AccountSnapshot{
		CashBalance:      cfg.StartingCash(),
		BuyingPower:      cfg.StartingCash(),
		PositionQuantity: 0,
		HasOpenOrder:     false,
	}
	replayState := replay.NewState(cfg.Symbol)

	lookbackBars := cfg.Scalping.BreakoutLookbackBars
	if lookbackBars <= 0 {
		lookbackBars = 1
	}

	var (
		decisions      []decisionPoint
		pending        *pendingOrder
		lastSnapshot   tradingstrategy.MarketSnapshot
		highSinceEntry float64
		lastStopLossAt *time.Time
		recentHighs    []float64 // circular buffer of recent highs
		recentLows     []float64 // circular buffer of recent lows
	)

	for _, event := range events {
		snapshot := replayState.Apply(event)
		lastSnapshot = snapshot

		if pending != nil && !event.At.Before(pending.FillAt) {
			prevPos := account.PositionQuantity
			wasStop := pending.Reason == "trailing stop triggered"
			applyPendingFill(pending, snapshot, event, &account, &decisions, cfg.BidAskSpreadPct)
			if account.PositionQuantity > prevPos {
				highSinceEntry = account.EntryPrice
			} else if account.PositionQuantity < prevPos {
				highSinceEntry = 0
				if wasStop {
					t := event.At
					lastStopLossAt = &t
				}
			}
			pending = nil
		}

		input := tradingstrategy.NewEvaluateInput(snapshot, account)

		// Track N-bar high/low for lookback-based breakout entries (daily/weekly strategies).
		// Important: evaluate against prior bars only (exclude current bar), then append current.
		if input.Price > 0 {
			if len(recentHighs) > 0 {
				maxHigh := recentHighs[0]
				minLow := recentLows[0]
				for _, h := range recentHighs {
					if h > maxHigh {
						maxHigh = h
					}
				}
				for _, l := range recentLows {
					if l < minLow {
						minLow = l
					}
				}
				input.LookbackHighPrice = maxHigh
				input.LookbackLowPrice = minLow
			}

			recentHighs = append(recentHighs, input.Price)
			recentLows = append(recentLows, input.Price)
			if len(recentHighs) > lookbackBars {
				recentHighs = recentHighs[len(recentHighs)-lookbackBars:]
				recentLows = recentLows[len(recentLows)-lookbackBars:]
			}
		}

		// Track trailing high while in position.
		if account.PositionQuantity > 0 && input.Price > highSinceEntry {
			highSinceEntry = input.Price
		}
		input.HighSinceEntry = highSinceEntry
		input.LastStopLossAt = lastStopLossAt
		if v, ok := rsiByTs[event.At.Unix()]; ok {
			value := v
			input.RSI = &value
		}
		if v, ok := macdByTs[event.At.Unix()]; ok {
			value := v
			input.MACD = &value
		}
		if v, ok := macdSignalByTs[event.At.Unix()]; ok {
			value := v
			input.MACDSignal = &value
		}
		if v, ok := bollUpperByTs[event.At.Unix()]; ok {
			value := v
			input.BollUpper = &value
		}
		if v, ok := bollMiddleByTs[event.At.Unix()]; ok {
			value := v
			input.BollMiddle = &value
		}
		if v, ok := bollLowerByTs[event.At.Unix()]; ok {
			value := v
			input.BollLower = &value
		}
		if input.BollUpper != nil && input.BollMiddle != nil && input.BollLower != nil && *input.BollMiddle != 0 {
			value := (*input.BollUpper - *input.BollLower) / *input.BollMiddle
			input.BollWidthPct = &value
		}
		decision := strategy.Evaluate(input)

		if decision.Action == tradingstrategy.ActionNone {
			continue
		}
		if pending != nil {
			continue
		}
		qty := decision.Quantity
		switch decision.Action {
		case tradingstrategy.ActionBuy:
			if qty <= 0 {
				continue
			}
		case tradingstrategy.ActionSell:
			if qty <= 0 || qty > account.PositionQuantity {
				qty = account.PositionQuantity
			}
			if qty <= 0 {
				continue
			}
		default:
			continue
		}
		pending = &pendingOrder{
			Action:   decision.Action,
			Quantity: qty,
			Reason:   decision.Reason,
			FillAt:   event.At.Add(cfg.FillLatency()),
		}
		account.HasOpenOrder = true
		if !event.At.Before(pending.FillAt) {
			prevPos := account.PositionQuantity
			wasStop := pending.Reason == "trailing stop triggered"
			applyPendingFill(pending, snapshot, event, &account, &decisions, cfg.BidAskSpreadPct)
			if account.PositionQuantity > prevPos {
				highSinceEntry = account.EntryPrice
			} else if account.PositionQuantity < prevPos {
				highSinceEntry = 0
				if wasStop {
					t := event.At
					lastStopLossAt = &t
				}
			}
			pending = nil
		}
	}

	// Fill any remaining pending order at end of data using the last known
	// snapshot, avoiding a redundant Apply that would double-update session state.
	if pending != nil && len(events) > 0 {
		lastEvent := events[len(events)-1]
		applyPendingFill(pending, lastSnapshot, lastEvent, &account, &decisions, cfg.BidAskSpreadPct)
	}

	lastPrice := prices[len(prices)-1].Close
	endingValue := account.CashBalance + (account.PositionQuantity * lastPrice)
	startingCash := cfg.StartingCash()
	return result{
		Symbol:        cfg.Symbol,
		Strategy:      cfg.Strategy,
		StartingCash:  startingCash,
		EndingCash:    account.CashBalance,
		EndingValue:   endingValue,
		TotalReturn:   (endingValue - startingCash) / startingCash,
		Prices:        prices,
		Decisions:     decisions,
		FinalPosition: account.PositionQuantity,
	}
}

func applyPendingFill(pending *pendingOrder, snapshot tradingstrategy.MarketSnapshot, event replay.Event, account *tradingstrategy.AccountSnapshot, decisions *[]decisionPoint, bidAskSpreadPct float64) {
	price := fillPrice(pending.Action, snapshot, event)
	if price <= 0 {
		return
	}
	account.HasOpenOrder = false
	switch pending.Action {
	case tradingstrategy.ActionBuy:
		qty := pending.Quantity
		// Deduct bid-ask spread on buy (price goes up by spread/2)
		effectivePrice := price * (1 + bidAskSpreadPct/2)
		cost := qty * effectivePrice
		if cost > account.CashBalance {
			qty = math.Floor(account.CashBalance / effectivePrice)
			cost = qty * effectivePrice
		}
		if qty <= 0 {
			return
		}
		account.CashBalance -= cost
		account.BuyingPower = account.CashBalance
		account.PositionQuantity += qty
		account.EntryPrice = effectivePrice
		*decisions = append(*decisions, decisionPoint{
			At:       event.At,
			Price:    price,
			Action:   tradingstrategy.ActionBuy,
			Quantity: qty,
			Reason:   pending.Reason,
		})
	case tradingstrategy.ActionSell:
		qty := pending.Quantity
		if qty > account.PositionQuantity {
			qty = account.PositionQuantity
		}
		if qty <= 0 {
			return
		}
		// Deduct bid-ask spread on sell (price goes down by spread/2)
		effectivePrice := price * (1 - bidAskSpreadPct/2)
		proceeds := qty * effectivePrice
		account.CashBalance += proceeds
		account.BuyingPower = account.CashBalance
		account.PositionQuantity -= qty
		account.EntryPrice = 0
		*decisions = append(*decisions, decisionPoint{
			At:       event.At,
			Price:    price,
			Action:   tradingstrategy.ActionSell,
			Quantity: qty,
			Reason:   pending.Reason,
		})
	}
}

// fillPrice returns the execution price for a pending order. When quote data is
// available it uses direction-aware pricing: buys fill at the ask, sells fill
// at the bid — matching how a market order would cross the spread in practice.
func fillPrice(action tradingstrategy.Action, snapshot tradingstrategy.MarketSnapshot, event replay.Event) float64 {
	switch action {
	case tradingstrategy.ActionBuy:
		if snapshot.AskPrice != nil {
			return *snapshot.AskPrice
		}
	case tradingstrategy.ActionSell:
		if snapshot.BidPrice != nil {
			return *snapshot.BidPrice
		}
	}
	switch {
	case snapshot.LastTradePrice != nil:
		return *snapshot.LastTradePrice
	case snapshot.BidPrice != nil && snapshot.AskPrice != nil:
		return (*snapshot.BidPrice + *snapshot.AskPrice) / 2
	case snapshot.BidPrice != nil:
		return *snapshot.BidPrice
	case snapshot.AskPrice != nil:
		return *snapshot.AskPrice
	case event.Trade != nil:
		return event.Trade.Price
	default:
		return 0
	}
}

type indicatorPoint struct {
	At    time.Time
	Value float64
}

func computeRSI(prices []replay.PricePoint, period int) []indicatorPoint {
	if len(prices) <= period || period < 2 {
		return nil
	}
	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		delta := prices[i].Close - prices[i-1].Close
		if delta > 0 {
			gainSum += delta
		} else {
			lossSum -= delta
		}
	}
	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)
	out := make([]indicatorPoint, 0, len(prices)-period)
	out = append(out, indicatorPoint{At: prices[period].At, Value: rsiFromAverages(avgGain, avgLoss)})
	for i := period + 1; i < len(prices); i++ {
		delta := prices[i].Close - prices[i-1].Close
		gain := 0.0
		loss := 0.0
		if delta > 0 {
			gain = delta
		} else {
			loss = -delta
		}
		avgGain = ((avgGain * float64(period-1)) + gain) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + loss) / float64(period)
		out = append(out, indicatorPoint{At: prices[i].At, Value: rsiFromAverages(avgGain, avgLoss)})
	}
	return out
}

func rsiFromAverages(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func computeMACD(prices []replay.PricePoint, fastPeriod int, slowPeriod int, signalPeriod int) ([]indicatorPoint, []indicatorPoint) {
	if len(prices) == 0 || fastPeriod < 2 || slowPeriod < 2 || signalPeriod < 2 || slowPeriod <= fastPeriod {
		return nil, nil
	}
	fastK := 2.0 / (float64(fastPeriod) + 1)
	slowK := 2.0 / (float64(slowPeriod) + 1)
	signalK := 2.0 / (float64(signalPeriod) + 1)
	fastEMA := prices[0].Close
	slowEMA := prices[0].Close
	macdSeries := make([]indicatorPoint, 0, len(prices))
	signalSeries := make([]indicatorPoint, 0, len(prices))
	var signalEMA float64
	hasSignal := false
	for i, p := range prices {
		if i > 0 {
			fastEMA = ((p.Close - fastEMA) * fastK) + fastEMA
			slowEMA = ((p.Close - slowEMA) * slowK) + slowEMA
		}
		macd := fastEMA - slowEMA
		macdSeries = append(macdSeries, indicatorPoint{At: p.At, Value: macd})
		if !hasSignal {
			signalEMA = macd
			hasSignal = true
		} else {
			signalEMA = ((macd - signalEMA) * signalK) + signalEMA
		}
		signalSeries = append(signalSeries, indicatorPoint{At: p.At, Value: signalEMA})
	}
	return macdSeries, signalSeries
}

func computeBollingerBands(prices []replay.PricePoint, period int, stdDevMultiplier float64) ([]indicatorPoint, []indicatorPoint, []indicatorPoint) {
	if len(prices) < period || period < 2 || stdDevMultiplier <= 0 {
		return nil, nil, nil
	}
	upper := make([]indicatorPoint, 0, len(prices)-period+1)
	middle := make([]indicatorPoint, 0, len(prices)-period+1)
	lower := make([]indicatorPoint, 0, len(prices)-period+1)
	windowSum := 0.0
	windowSqSum := 0.0
	for i := 0; i < len(prices); i++ {
		close := prices[i].Close
		windowSum += close
		windowSqSum += close * close
		if i >= period {
			out := prices[i-period].Close
			windowSum -= out
			windowSqSum -= out * out
		}
		if i < period-1 {
			continue
		}
		mean := windowSum / float64(period)
		variance := (windowSqSum / float64(period)) - (mean * mean)
		if variance < 0 {
			variance = 0
		}
		stddev := math.Sqrt(variance)
		at := prices[i].At
		middle = append(middle, indicatorPoint{At: at, Value: mean})
		upper = append(upper, indicatorPoint{At: at, Value: mean + (stdDevMultiplier * stddev)})
		lower = append(lower, indicatorPoint{At: at, Value: mean - (stdDevMultiplier * stddev)})
	}
	return upper, middle, lower
}

func filterIndicatorSeriesToRange(points []indicatorPoint, start time.Time, end time.Time) []indicatorPoint {
	out := make([]indicatorPoint, 0, len(points))
	for _, p := range points {
		if p.At.Before(start) || p.At.After(end) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func filterIndicatorToMarketHours(points []indicatorPoint, tz *time.Location, timeframe string) []indicatorPoint {
	// For daily and weekly timeframes, don't filter to market hours (they need end-of-day/week closes).
	// Only filter intraday (1Min, 5Min, etc.) to 9:30 AM - 4:00 PM.
	if timeframe == "1Day" || timeframe == "1Week" {
		return points
	}
	out := make([]indicatorPoint, 0, len(points))
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

func chartIndicatorPoints(points []indicatorPoint) []chart.IndicatorPoint {
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

func chartDecisions(decisions []decisionPoint) []chart.DecisionMarker {
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
