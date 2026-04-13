package tradingstrategy

import "fmt"

// ProfitTargetExitStrategy exits when price reaches or exceeds
// entryPrice + ProfitTargetMultiplier × ATR, providing an explicit
// take-profit level that scales with current volatility.
//
// Setting the multiplier above the ATR stop multiplier ensures a positive
// risk:reward ratio (e.g. stop at 2×ATR, target at 3×ATR gives 1.5R).
//
// Missing ATR data causes the strategy to abstain rather than silently
// pass. When ProfitTargetMultiplier is zero the strategy is disabled.
type ProfitTargetExitStrategy struct {
	profitTargetMultiplier float64
}

type NewProfitTargetExitStrategyInput struct {
	ProfitTargetMultiplier float64 // ATR multiple above entry price; 0 disables
}

func NewProfitTargetExitStrategy(input NewProfitTargetExitStrategyInput) *ProfitTargetExitStrategy {
	return &ProfitTargetExitStrategy{profitTargetMultiplier: input.ProfitTargetMultiplier}
}

func (strategy *ProfitTargetExitStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 || strategy.profitTargetMultiplier <= 0 || input.EntryPrice <= 0 {
		return Decision{Action: ActionNone}
	}
	if input.ATR == nil {
		return Decision{Action: ActionNone, Reason: "profit target: atr unavailable"}
	}
	target := input.EntryPrice + strategy.profitTargetMultiplier**input.ATR
	if input.Price < target {
		return Decision{
			Action: ActionNone,
			Reason: fmt.Sprintf("profit target: price %.2f below target %.2f (entry %.2f + %.1f×ATR %.2f)", input.Price, target, input.EntryPrice, strategy.profitTargetMultiplier, *input.ATR),
		}
	}
	return Decision{
		Action:   ActionSell,
		Reason:   fmt.Sprintf("profit target: price %.2f at or above target %.2f (entry %.2f + %.1f×ATR %.2f)", input.Price, target, input.EntryPrice, strategy.profitTargetMultiplier, *input.ATR),
		Quantity: input.PositionQuantity,
	}
}
