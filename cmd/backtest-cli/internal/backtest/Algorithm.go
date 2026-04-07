package backtest

import (
	"fmt"
	"math"
	"time"

	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/backtestconfig"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/indicator"
	"github.com/kduong/trading-backend/cmd/backtest-cli/internal/replay"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
)

type pendingOrder struct {
	Action   tradingstrategy.Action
	Quantity float64
	Reason   string
	FillAt   time.Time
}

type result struct {
	Symbol        string
	StartingCash  float64
	EndingCash    float64
	EndingValue   float64
	TotalReturn   float64
	Prices        []replay.PricePoint
	Decisions     []DecisionPoint
	FinalPosition float64
	SharpeRatio   float64
	WinRate       float64
	TradeCount    int
}

// Run simulates a backtest over the given prices and events.
// indicatorPrices should include warmup bars before the backtest range so that
// EMAs and other indicators are fully converged by the time the simulation begins.
func Run(cfg backtestconfig.Config, prices []replay.PricePoint, indicatorPrices []replay.PricePoint, events []replay.Event) result {
	if len(indicatorPrices) == 0 {
		indicatorPrices = prices
	}
	warnIfInsufficientWarmup(len(indicatorPrices), cfg)
	strategy := tradingstrategy.FromParameters(&cfg.TradingParameters)
	rsiSeries := indicator.ComputeRSI(indicatorPrices, cfg.Indicators.RSIPeriod)
	macdSeries, macdSignalSeries := indicator.ComputeMACD(indicatorPrices, cfg.Indicators.MACDFastPeriod, cfg.Indicators.MACDSlowPeriod, cfg.Indicators.MACDSignalPeriod)
	bollUpperSeries, bollMiddleSeries, bollLowerSeries := indicator.ComputeBollingerBands(indicatorPrices, cfg.Indicators.BollingerPeriod, cfg.Indicators.BollingerStdDev)
	smaSeries := indicator.ComputeSMA(indicatorPrices, cfg.Indicators.SMAPeriod)
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
	smaByTs := make(map[int64]float64, len(smaSeries))
	for _, p := range smaSeries {
		smaByTs[p.At.Unix()] = p.Value
	}
	account := tradingstrategy.AccountSnapshot{
		CashBalance:      cfg.StartingCash(),
		BuyingPower:      cfg.StartingCash(),
		PositionQuantity: 0,
		HasOpenOrder:     false,
	}
	replayState := replay.NewState(cfg.Symbol)

	lookbackBars := cfg.TradingParameters.BreakoutLookbackBars
	if lookbackBars <= 0 {
		lookbackBars = 1
	}

	var (
		decisions      []DecisionPoint
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
		if v, ok := smaByTs[event.At.Unix()]; ok {
			value := v
			input.SMA = &value
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
	returns := computeTradeReturns(decisions)
	return result{
		Symbol:        cfg.Symbol,
		StartingCash:  startingCash,
		EndingCash:    account.CashBalance,
		EndingValue:   endingValue,
		TotalReturn:   (endingValue - startingCash) / startingCash,
		Prices:        prices,
		Decisions:     decisions,
		FinalPosition: account.PositionQuantity,
		SharpeRatio:   computeSharpe(returns),
		WinRate:       computeWinRate(returns),
		TradeCount:    len(returns),
	}
}

// warnIfInsufficientWarmup prints a warning when indicatorBars is fewer than
// the minimum required for each indicator to be fully converged. This commonly
// happens with recently listed stocks (IPO/spin-off) where historical data
// before the backtest start date simply does not exist.
func warnIfInsufficientWarmup(indicatorBars int, cfg backtestconfig.Config) {
	ind := cfg.Indicators
	checks := []struct {
		name    string
		minBars int
	}{
		{"RSI(" + fmt.Sprintf("%d", ind.RSIPeriod) + ")", ind.RSIPeriod + 1},
		{fmt.Sprintf("MACD(%d,%d,%d)", ind.MACDFastPeriod, ind.MACDSlowPeriod, ind.MACDSignalPeriod), ind.MACDSlowPeriod + ind.MACDSignalPeriod},
		{"Bollinger(" + fmt.Sprintf("%d", ind.BollingerPeriod) + ")", ind.BollingerPeriod},
		{"SMA(" + fmt.Sprintf("%d", ind.SMAPeriod) + ")", ind.SMAPeriod},
	}
	for _, c := range checks {
		if indicatorBars < c.minBars {
			fmt.Printf("WARNING: only %d indicator bars available, %s needs %d — indicator will be incomplete (IPO/spin-off with limited history?)\n",
				indicatorBars, c.name, c.minBars)
		}
	}
}

// computeTradeReturns pairs up buy/sell decisions and returns per-trade returns.
func computeTradeReturns(decisions []DecisionPoint) []float64 {
	var returns []float64
	for i := 0; i+1 < len(decisions); i += 2 {
		buy := decisions[i]
		sell := decisions[i+1]
		if buy.Action != tradingstrategy.ActionBuy || sell.Action != tradingstrategy.ActionSell {
			continue
		}
		if buy.Price <= 0 {
			continue
		}
		ret := (sell.Price - buy.Price) / buy.Price
		returns = append(returns, ret)
	}
	return returns
}

func computeSharpe(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))
	var variance float64
	for _, r := range returns {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(returns))
	stddev := math.Sqrt(variance)
	if stddev == 0 {
		return 0
	}
	return mean / stddev
}

func computeWinRate(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	var wins int
	for _, r := range returns {
		if r > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(returns))
}

func applyPendingFill(pending *pendingOrder, snapshot tradingstrategy.MarketSnapshot, event replay.Event, account *tradingstrategy.AccountSnapshot, decisions *[]DecisionPoint, bidAskSpreadPct float64) {
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
		*decisions = append(*decisions, DecisionPoint{
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
		*decisions = append(*decisions, DecisionPoint{
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
