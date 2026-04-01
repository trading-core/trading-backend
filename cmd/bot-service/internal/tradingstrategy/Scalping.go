package tradingstrategy

import (
	"math"
	"time"
)

// ScalpingConfig holds tunable parameters for the scalping strategy.
// MaxPositionFraction is the fraction of buying power to deploy per trade (e.g. 0.1 = 10%).
// TakeProfitPct is the percentage gain above entry price to trigger a profit exit (e.g. 0.005 = 0.5%).
// SessionStart and SessionEnd define the window (hour in exchange local time) during which new entries are allowed.
type ScalpingConfig struct {
	MaxPositionFraction float64
	TakeProfitPct       float64
	SessionStart        int // hour 0-23
	SessionEnd          int // hour 0-23, exclusive
}

var defaultScalpingConfig = ScalpingConfig{
	MaxPositionFraction: 0.1,
	TakeProfitPct:       0.005,
	SessionStart:        9,
	SessionEnd:          15,
}

var usMarketLocation = loadUSMarketLocation()

func loadUSMarketLocation() *time.Location {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.Local
	}
	return location
}

type Scalping struct {
	Config     ScalpingConfig
	entryPrice float64
}

func NewScalping(cfg ScalpingConfig) *Scalping {
	return &Scalping{Config: cfg}
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
	// --- Exit logic (evaluated before entry so we don't ignore an open position) ---
	if input.PositionQuantity > 0 {
		// Take-profit: price reached target above entry
		if strategy.entryPrice > 0 && input.Price >= strategy.entryPrice*(1+strategy.Config.TakeProfitPct) {
			strategy.entryPrice = 0
			return Decision{
				Action:   ActionSell,
				Reason:   "take-profit target reached",
				Quantity: input.PositionQuantity,
			}
		}
		// Stop-loss: price fell back below session open
		if input.Price < input.SessionOpenPrice {
			strategy.entryPrice = 0
			return Decision{
				Action:   ActionSell,
				Reason:   "price lost session open",
				Quantity: input.PositionQuantity,
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
	hour := input.Now.In(usMarketLocation).Hour()
	if hour < strategy.Config.SessionStart || hour >= strategy.Config.SessionEnd {
		return Decision{Action: ActionNone, Reason: "outside trading session window"}
	}
	// Breakout entry: price breaks above session high
	if input.Price > input.SessionHighPrice {
		maxCapital := buyingPower * strategy.Config.MaxPositionFraction
		qty := math.Floor(maxCapital / input.Price)
		if qty < 1 {
			return Decision{Action: ActionNone, Reason: "insufficient buying power for one share"}
		}
		strategy.entryPrice = input.Price
		return Decision{
			Action:   ActionBuy,
			Reason:   "price broke above session high",
			Quantity: qty,
		}
	}
	return Decision{Action: ActionNone, Reason: "no momentum breakout signal"}
}
