package tradingstrategy

import (
	"math"
	"time"
)

// Scalping strategy holds parameters.
// MaxPositionFraction is the fraction of buying power to deploy per trade (e.g. 0.1 = 10%).
// TakeProfitPct is the percentage gain above entry price to trigger a profit exit (e.g. 0.005 = 0.5%).
// StopLossPct is the trailing stop-loss percentage below the highest price since entry (e.g. 0.02 = 2%).
// SessionStart and SessionEnd define the window (hour in exchange local time) during which new entries are allowed.
// Positions are force-closed when the hour reaches SessionEnd.

type Scalping struct {
	EntryMode                string // "breakout" or "pullback"
	MaxPositionFraction      float64
	TakeProfitPct            float64
	StopLossPct              float64
	SessionStart             int // hour 0-23
	SessionEnd               int // hour 0-23, exclusive
	MinRSI                   float64
	RequireMACDSignal        bool
	RequireBollingerBreakout bool
	MinBollingerWidthPct     float64
	RequireBollingerSqueeze  bool
	MaxBollingerWidthPct     float64
	ReentryCooldownMinutes   int
	UseVolatilityTP          bool
	VolatilityTPMultiplier   float64
	RiskPerTradePct          float64
	BreakoutLookbackBars     int // number of bars to lookback for breakout (1=session high, 5=5-bar high)
}

func NewScalping() *Scalping {
	return &Scalping{
		EntryMode:                "pullback",
		MaxPositionFraction:      0.1,
		TakeProfitPct:            0.005,
		StopLossPct:              0.02,
		SessionStart:             10,
		SessionEnd:               15,
		MinRSI:                   40,
		RequireMACDSignal:        true,
		RequireBollingerBreakout: false,
		MinBollingerWidthPct:     0,
		RequireBollingerSqueeze:  false,
		MaxBollingerWidthPct:     0.02,
		ReentryCooldownMinutes:   5,
		UseVolatilityTP:          false,
		VolatilityTPMultiplier:   0.5,
		RiskPerTradePct:          0,
		BreakoutLookbackBars:     1,
	}
}

func (strategy *Scalping) Type() StrategyType {
	return StrategyTypeScalping
}

func (strategy *Scalping) Evaluate(input EvaluateInput) Decision {
	if input.HasOpenOrder {
		return Decision{Action: ActionNone, Reason: "waiting for open order to resolve"}
	}
	if input.Price <= 0 {
		return Decision{Action: ActionNone, Reason: "price unavailable"}
	}

	hour := input.Now.In(USMarketLocation).Hour()

	// --- Exit logic (evaluated before entry so we don't ignore an open position) ---
	if input.PositionQuantity > 0 {
		// 1. Forced end-of-day exit: close position when session ends.
		if hour >= strategy.SessionEnd {
			return Decision{
				Action:   ActionSell,
				Reason:   "forced end-of-day exit",
				Quantity: input.PositionQuantity,
			}
		}

		// 2. Take-profit (possibly volatility-scaled via Bollinger width).
		if input.EntryPrice > 0 {
			effectiveTP := strategy.TakeProfitPct
			if strategy.UseVolatilityTP && input.BollWidthPct != nil {
				dynamicTP := *input.BollWidthPct * strategy.VolatilityTPMultiplier
				if dynamicTP > effectiveTP {
					effectiveTP = dynamicTP
				}
			}
			if input.Price >= input.EntryPrice*(1+effectiveTP) {
				return Decision{
					Action:   ActionSell,
					Reason:   "take-profit target reached",
					Quantity: input.PositionQuantity,
				}
			}
		}

		// 3. Trailing stop-loss: exit when price drops StopLossPct below
		//    the highest price observed since entry.
		if strategy.StopLossPct > 0 && input.EntryPrice > 0 {
			trailingHigh := input.EntryPrice
			if input.HighSinceEntry > trailingHigh {
				trailingHigh = input.HighSinceEntry
			}
			stopLevel := trailingHigh * (1 - strategy.StopLossPct)
			if input.Price <= stopLevel {
				return Decision{
					Action:   ActionSell,
					Reason:   "trailing stop triggered",
					Quantity: input.PositionQuantity,
				}
			}
		}

		return Decision{Action: ActionNone, Reason: "holding position"}
	}

	// --- Entry logic ---
	buyingPower := input.BuyingPower
	if buyingPower <= 0 {
		buyingPower = input.CashBalance
	}
	if buyingPower <= 0 {
		return Decision{Action: ActionNone, Reason: "no buying power available"}
	}

	// Time-of-day guard: only enter during configured US equities session window.
	if hour < strategy.SessionStart || hour >= strategy.SessionEnd {
		return Decision{Action: ActionNone, Reason: "outside trading session window"}
	}

	// Re-entry cooldown after a trailing-stop exit.
	if strategy.ReentryCooldownMinutes > 0 && input.LastStopLossAt != nil {
		cooldownEnd := input.LastStopLossAt.Add(time.Duration(strategy.ReentryCooldownMinutes) * time.Minute)
		if input.Now.Before(cooldownEnd) {
			return Decision{Action: ActionNone, Reason: "re-entry cooldown active"}
		}
	}

	// RSI filter.
	if input.RSI == nil {
		return Decision{Action: ActionNone, Reason: "rsi unavailable"}
	}
	if *input.RSI < strategy.MinRSI {
		return Decision{Action: ActionNone, Reason: "rsi below threshold"}
	}

	// MACD filter.
	if strategy.RequireMACDSignal {
		if input.MACD == nil || input.MACDSignal == nil {
			return Decision{Action: ActionNone, Reason: "macd unavailable"}
		}
		if *input.MACD <= *input.MACDSignal {
			return Decision{Action: ActionNone, Reason: "macd below signal"}
		}
	}

	// Bollinger breakout filter (price above upper band).
	if strategy.RequireBollingerBreakout {
		if input.BollUpper == nil || input.BollMiddle == nil || input.BollLower == nil {
			return Decision{Action: ActionNone, Reason: "bollinger unavailable"}
		}
		if input.Price <= *input.BollUpper {
			return Decision{Action: ActionNone, Reason: "price below upper bollinger"}
		}
		if strategy.MinBollingerWidthPct > 0 {
			if input.BollWidthPct == nil {
				return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
			}
			if *input.BollWidthPct < strategy.MinBollingerWidthPct {
				return Decision{Action: ActionNone, Reason: "bollinger width too narrow"}
			}
		}
	}

	// Bollinger squeeze filter (low volatility compression before breakout).
	if strategy.RequireBollingerSqueeze {
		if input.BollWidthPct == nil {
			return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
		}
		if *input.BollWidthPct >= strategy.MaxBollingerWidthPct {
			return Decision{Action: ActionNone, Reason: "bollinger not in squeeze"}
		}
	}

	// --- Entry trigger (mode-dependent) ---
	entryTriggered := false

	switch strategy.EntryMode {
	case "pullback":
		// Pullback entry: buy when price dips to or below the Bollinger middle
		// band while RSI/MACD still confirm upward momentum. This enters at
		// mean-reversion support rather than chasing breakouts at resistance.
		if input.BollMiddle == nil {
			return Decision{Action: ActionNone, Reason: "bollinger middle unavailable for pullback"}
		}
		if input.Price <= *input.BollMiddle {
			entryTriggered = true
		}

	default: // "breakout"
		// Breakout entry: price breaks above a reference high (session-based or lookback-based).
		// For 1-min scalping: use SessionHighPrice (resets daily, tracks intraday range).
		// For daily/weekly: use LookbackHighPrice (e.g., 5-bar high, avoids noisy daily resets).
		referenceHigh := input.SessionHighPrice
		if strategy.BreakoutLookbackBars > 1 && input.LookbackHighPrice > 0 {
			// Use N-bar high if configured and available
			referenceHigh = input.LookbackHighPrice
		}
		if referenceHigh > 0 && input.Price > referenceHigh {
			entryTriggered = true
		}
	}

	if !entryTriggered {
		return Decision{Action: ActionNone, Reason: "no entry signal"}
	}

	var qty float64
	if strategy.RiskPerTradePct > 0 && strategy.StopLossPct > 0 {
		riskAmount := buyingPower * strategy.RiskPerTradePct
		stopDistance := input.Price * strategy.StopLossPct
		qty = math.Floor(riskAmount / stopDistance)
		maxQty := math.Floor(buyingPower * strategy.MaxPositionFraction / input.Price)
		if qty > maxQty {
			qty = maxQty
		}
	} else {
		maxCapital := buyingPower * strategy.MaxPositionFraction
		qty = math.Floor(maxCapital / input.Price)
	}
	if qty < 1 {
		return Decision{Action: ActionNone, Reason: "insufficient buying power for one share"}
	}
	return Decision{
		Action:   ActionBuy,
		Reason:   "entry signal: " + strategy.EntryMode,
		Quantity: qty,
	}
}
