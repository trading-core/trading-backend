package tradingstrategy

import "fmt"

// ATRStopStrategy exits a position when price falls at or below
// highSinceEntry − ATRMultiplier × ATR. Using ATR rather than a fixed
// percentage means the stop distance adapts to current volatility: wider
// in choppy markets to avoid noise-driven exits, tighter in calm markets
// to lock in gains. The stop does not fire when ATR data is unavailable.
type ATRStopStrategy struct {
	atrMultiplier float64
}

type NewATRStopStrategyInput struct {
	ATRMultiplier float64 // e.g. 2.0; stop at highSinceEntry − ATRMultiplier × ATR; 0 disables
}

func NewATRStopStrategy(input NewATRStopStrategyInput) *ATRStopStrategy {
	return &ATRStopStrategy{
		atrMultiplier: input.ATRMultiplier,
	}
}

func (strategy *ATRStopStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 || strategy.atrMultiplier <= 0 || input.EntryPrice <= 0 {
		return Decision{Action: ActionNone}
	}
	if input.ATR == nil {
		return Decision{Action: ActionNone, Reason: "atr stop: atr unavailable"}
	}

	trailingHigh := input.HighSinceEntry
	if trailingHigh == 0 {
		trailingHigh = input.EntryPrice
	}
	stopLevel := trailingHigh - strategy.atrMultiplier**input.ATR
	if input.Price > stopLevel {
		return Decision{
			Action: ActionNone,
			Reason: fmt.Sprintf("atr stop: price %.2f above stop %.2f (high %.2f − %.1f×ATR %.2f)", input.Price, stopLevel, trailingHigh, strategy.atrMultiplier, *input.ATR),
		}
	}

	return Decision{
		Action:   ActionSell,
		Reason:   fmt.Sprintf("atr stop: price %.2f at or below stop %.2f (high %.2f − %.1f×ATR %.2f)", input.Price, stopLevel, trailingHigh, strategy.atrMultiplier, *input.ATR),
		Quantity: input.PositionQuantity,
	}
}
