package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/chart"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"
	"github.com/kduong/trading-backend/internal/broker/alpaca"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

const AlpacaStockBarLimit = 10000

type decisionPoint struct {
	At       time.Time
	Price    float64
	Action   tradingstrategy.Action
	Quantity float64
	Reason   string
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

type pendingOrder struct {
	Action   tradingstrategy.Action
	Quantity float64
	Reason   string
	FillAt   time.Time
}

func main() {
	ctx := context.Background()
	cash := config.EnvInt("BACKTEST_CASH", 20000)
	fatal.Unless(cash >= 0, "BACKTEST_CASH must be greater than zero")
	symbol := config.EnvString("BACKTEST_SYMBOL", "AAPL")
	strategyName := config.EnvString("BACKTEST_STRATEGY", "scalping")
	alpacaTF := config.EnvString("BACKTEST_ALPACA_TIMEFRAME", "1Min")
	alpacaStart := config.EnvString("BACKTEST_ALPACA_START", "")
	alpacaEnd := config.EnvString("BACKTEST_ALPACA_END", "")
	alpacaFeed := config.EnvString("BACKTEST_ALPACA_FEED", "iex")
	replayEventsFile := config.EnvString("BACKTEST_REPLAY_EVENTS_FILE", "")
	fillLatencyMS := config.EnvInt("BACKTEST_FILL_LATENCY_MS", 0)
	fatal.Unless(fillLatencyMS >= 0, "BACKTEST_FILL_LATENCY_MS must be non-negative")
	bidAskSpreadPct := config.EnvFloat64("BACKTEST_BID_ASK_SPREAD_PCT", 0)
	fatal.Unless(bidAskSpreadPct >= 0, "BACKTEST_BID_ASK_SPREAD_PCT must be non-negative")
	err := tradingstrategy.ValidateType(strategyName)
	fatal.OnError(err)
	strategyParams := tradingstrategy.ScalpingParams{
		MaxPositionFraction: config.EnvFloat64("BACKTEST_MAX_POSITION_FRACTION", 0),
		TakeProfitPct:       config.EnvFloat64("BACKTEST_TAKE_PROFIT_PCT", 0),
		SessionStart:        config.EnvInt("BACKTEST_SESSION_START", 0),
		SessionEnd:          config.EnvInt("BACKTEST_SESSION_END", 0),
	}
	sweep := config.EnvBool("BACKTEST_SWEEP", false)

	var (
		prices []replay.PricePoint
		events []replay.Event
	)
	if replayEventsFile != "" {
		events, err = replay.LoadEventsFromFile(replayEventsFile, symbol)
		fatal.OnError(err)
		fatal.Unlessf(len(events) > 0, "replay file returned no events (path=%s symbol=%s)", replayEventsFile, symbol)
		prices = replay.PriceSeries(events)
		fatal.Unlessf(len(prices) > 0, "replay file returned no plottable prices (path=%s symbol=%s)", replayEventsFile, symbol)
	} else {
		prices, err = loadCandlesFromAlpaca(ctx, loadCandlesFromAlpacaInput{
			Symbol:    symbol,
			Timeframe: alpacaTF,
			Limit:     AlpacaStockBarLimit,
			Start:     alpacaStart,
			End:       alpacaEnd,
			Feed:      alpacaFeed,
		})
		fatal.OnError(err)
		fatal.Unlessf(len(prices) > 0, "alpaca returned no candle rows (symbol=%s timeframe=%s start=%q end=%q feed=%s limit=%d)", symbol, alpacaTF, alpacaStart, alpacaEnd, alpacaFeed, AlpacaStockBarLimit)
		events = replay.EventsFromCandles(symbol, prices)
	}

	fillLatency := time.Duration(fillLatencyMS) * time.Millisecond

	outputDir := fmt.Sprintf("./tmp/%s", symbol)
	err = os.MkdirAll(outputDir, 0o755)
	fatal.OnError(err)

	if sweep {
		runSweep(symbol, strategyName, float64(cash), prices, events, fillLatency, bidAskSpreadPct, outputDir)
		return
	}

	backTestResult := runBacktest(symbol, strategyName, strategyParams, float64(cash), prices, events, fillLatency, bidAskSpreadPct)
	outputPNG := fmt.Sprintf("%s/backtest.png", outputDir)
	err = chart.Render(chart.RenderInput{
		Symbol:      backTestResult.Symbol,
		Strategy:    backTestResult.Strategy,
		TotalReturn: backTestResult.TotalReturn,
		Prices:      chartPrices(backTestResult.Prices),
		Decisions:   chartDecisions(backTestResult.Decisions),
		Timezone:    tradingstrategy.USMarketLocation,
	}, outputPNG)
	fatal.OnError(err)

	fmt.Printf("Backtest complete for %s (%s)\n", backTestResult.Symbol, backTestResult.Strategy)
	fmt.Printf("Rows: %d\n", len(backTestResult.Prices))
	fmt.Printf("Decisions: %d\n", len(backTestResult.Decisions))
	fmt.Printf("Starting cash: %.2f\n", backTestResult.StartingCash)
	fmt.Printf("Ending cash: %.2f\n", backTestResult.EndingCash)
	fmt.Printf("Ending value: %.2f\n", backTestResult.EndingValue)
	fmt.Printf("Total return: %.2f%%\n", backTestResult.TotalReturn*100)
	fmt.Printf("Output image: %s\n", outputPNG)
}

func runSweep(symbol string, strategyName string, startingCash float64, prices []replay.PricePoint, events []replay.Event, fillLatency time.Duration, bidAskSpreadPct float64, outputDir string) {
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
						params := tradingstrategy.ScalpingParams{
							TakeProfitPct:       tp,
							MaxPositionFraction: pos,
							SessionStart:        ss,
							SessionEnd:          se,
						}
						var totalReturn float64
						var winWindows, totalTrades, windows int
						for i := 0; i+ws <= len(days); i++ {
							windowEvents, windowPrices := mergeWindow(days[i : i+ws])
							res := runBacktest(symbol, strategyName, params, startingCash, windowPrices, windowEvents, fillLatency, bidAskSpreadPct)
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

func runBacktest(symbol string, strategyName string, params tradingstrategy.ScalpingParams, startingCash float64, prices []replay.PricePoint, events []replay.Event, fillLatency time.Duration, bidAskSpreadPct float64) result {
	strategy := tradingstrategy.NewWithParams(strategyName, params)
	account := tradingstrategy.AccountSnapshot{
		CashBalance:      startingCash,
		BuyingPower:      startingCash,
		PositionQuantity: 0,
		HasOpenOrder:     false,
	}
	replayState := replay.NewState(symbol)

	var (
		decisions    []decisionPoint
		pending      *pendingOrder
		lastSnapshot tradingstrategy.MarketSnapshot
	)

	for _, event := range events {
		snapshot := replayState.Apply(event)
		lastSnapshot = snapshot

		if pending != nil && !event.At.Before(pending.FillAt) {
			applyPendingFill(pending, snapshot, event, &account, &decisions, bidAskSpreadPct)
			pending = nil
		}

		input := tradingstrategy.NewEvaluateInput(snapshot, account)
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
			FillAt:   event.At.Add(fillLatency),
		}
		account.HasOpenOrder = true
		if !event.At.Before(pending.FillAt) {
			applyPendingFill(pending, snapshot, event, &account, &decisions, bidAskSpreadPct)
			pending = nil
		}
	}

	// Fill any remaining pending order at end of data using the last known
	// snapshot, avoiding a redundant Apply that would double-update session state.
	if pending != nil && len(events) > 0 {
		lastEvent := events[len(events)-1]
		applyPendingFill(pending, lastSnapshot, lastEvent, &account, &decisions, bidAskSpreadPct)
	}

	lastPrice := prices[len(prices)-1].Close
	endingValue := account.CashBalance + (account.PositionQuantity * lastPrice)
	return result{
		Symbol:        symbol,
		Strategy:      strategyName,
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

type loadCandlesFromAlpacaInput struct {
	Symbol    string
	Timeframe string
	Limit     int
	Start     string
	End       string
	Feed      string
}

func loadCandlesFromAlpaca(ctx context.Context, input loadCandlesFromAlpacaInput) ([]replay.PricePoint, error) {
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

	points := make([]replay.PricePoint, 0, len(barsOutput.Bars))
	for _, bar := range barsOutput.Bars {
		at, err := parseTimestamp(bar.Time)
		if err != nil {
			return nil, fmt.Errorf("invalid alpaca bar time %q: %w", bar.Time, err)
		}
		points = append(points, replay.PricePoint{At: at, Close: bar.Close})
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
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
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
