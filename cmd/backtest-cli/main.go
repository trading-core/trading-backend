package main

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	cash := config.EnvInt("BACKTEST_CASH", 10000)
	fatal.Unless(cash >= 0, "BACKTEST_CASH must be greater than zero")
	alpacaLimit := config.EnvInt("BACKTEST_ALPACA_LIMIT", 1000)
	fatal.Unless(alpacaLimit >= 0, "BACKTEST_ALPACA_LIMIT must be greater than zero")
	symbol := config.EnvString("BACKTEST_SYMBOL", "AAPL")
	strategyName := config.EnvString("BACKTEST_STRATEGY", "scalping")
	alpacaTF := config.EnvString("BACKTEST_ALPACA_TIMEFRAME", "1Min")
	alpacaStart := config.EnvString("BACKTEST_ALPACA_START", "")
	alpacaEnd := config.EnvString("BACKTEST_ALPACA_END", "")
	alpacaFeed := config.EnvString("BACKTEST_ALPACA_FEED", "iex")
	replayEventsFile := config.EnvString("BACKTEST_REPLAY_EVENTS_FILE", "")
	fillLatencyMS := config.EnvInt("BACKTEST_FILL_LATENCY_MS", 0)
	fatal.Unless(fillLatencyMS >= 0, "BACKTEST_FILL_LATENCY_MS must be non-negative")
	outputPNG := config.EnvString("BACKTEST_OUTPUT", "backtest.png")
	err := tradingstrategy.ValidateType(strategyName)
	fatal.OnError(err)

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
			Limit:     alpacaLimit,
			Start:     alpacaStart,
			End:       alpacaEnd,
			Feed:      alpacaFeed,
		})
		fatal.OnError(err)
		fatal.Unlessf(len(prices) > 0, "alpaca returned no candle rows (symbol=%s timeframe=%s start=%q end=%q feed=%s limit=%d)", symbol, alpacaTF, alpacaStart, alpacaEnd, alpacaFeed, alpacaLimit)
		events = replay.EventsFromCandles(symbol, prices)
	}

	res := runBacktest(symbol, strategyName, float64(cash), prices, events, time.Duration(fillLatencyMS)*time.Millisecond)
	err = chart.Render(chart.RenderInput{
		Symbol:      res.Symbol,
		Strategy:    res.Strategy,
		TotalReturn: res.TotalReturn,
		Prices:      chartPrices(res.Prices),
		Decisions:   chartDecisions(res.Decisions),
		Timezone:    tradingstrategy.USMarketLocation,
	}, outputPNG)
	fatal.OnError(err)

	fmt.Printf("Backtest complete for %s (%s)\n", res.Symbol, res.Strategy)
	fmt.Printf("Rows: %d\n", len(res.Prices))
	fmt.Printf("Decisions: %d\n", len(res.Decisions))
	fmt.Printf("Starting cash: %.2f\n", res.StartingCash)
	fmt.Printf("Ending cash: %.2f\n", res.EndingCash)
	fmt.Printf("Ending value: %.2f\n", res.EndingValue)
	fmt.Printf("Total return: %.2f%%\n", res.TotalReturn*100)
	fmt.Printf("Output image: %s\n", outputPNG)
}

func runBacktest(symbol string, strategyName string, startingCash float64, prices []replay.PricePoint, events []replay.Event, fillLatency time.Duration) result {
	strategy := tradingstrategy.New(strategyName)
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
			applyPendingFill(pending, snapshot, event, &account, &decisions)
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
			applyPendingFill(pending, snapshot, event, &account, &decisions)
			pending = nil
		}
	}

	// Fill any remaining pending order at end of data using the last known
	// snapshot, avoiding a redundant Apply that would double-update session state.
	if pending != nil && len(events) > 0 {
		lastEvent := events[len(events)-1]
		applyPendingFill(pending, lastSnapshot, lastEvent, &account, &decisions)
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

func applyPendingFill(pending *pendingOrder, snapshot tradingstrategy.MarketSnapshot, event replay.Event, account *tradingstrategy.AccountSnapshot, decisions *[]decisionPoint) {
	price := fillPrice(pending.Action, snapshot, event)
	if price <= 0 {
		return
	}
	account.HasOpenOrder = false
	switch pending.Action {
	case tradingstrategy.ActionBuy:
		qty := pending.Quantity
		cost := qty * price
		if cost > account.CashBalance {
			qty = math.Floor(account.CashBalance / price)
			cost = qty * price
		}
		if qty <= 0 {
			return
		}
		account.CashBalance -= cost
		account.BuyingPower = account.CashBalance
		account.PositionQuantity += qty
		account.EntryPrice = price
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
		proceeds := qty * price
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
