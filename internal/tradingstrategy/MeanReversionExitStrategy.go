package tradingstrategy

// MeanReversionExitStrategy exits a range-mode position when price reaches
// or exceeds the Bollinger middle band, indicating that the mean-reversion
// trade has hit its target.
//
// The middle band represents the statistical mean around which price was
// expected to revert after an oversold entry at the lower band. Exiting at
// the mean avoids giving back unrealised gains by waiting for an overbought
// reading that may not materialise in a ranging market.
//
// To avoid contaminating trend-mode positions, the exit only fires when the
// entry price was below the current Bollinger middle band. Trend entries
// occur near or above the middle band (price above SMA ≈ BollMiddle), so
// this guard naturally limits the exit to range-mode entries without
// requiring knowledge of the original entry strategy.
//
// Missing BollMiddle data causes the strategy to abstain rather than
// silently pass. When Enabled is false the strategy is disabled.
type MeanReversionExitStrategy struct {
	enabled bool
}

type NewMeanReversionExitStrategyInput struct {
	Enabled bool // true to enable; false disables
}

func NewMeanReversionExitStrategy(input NewMeanReversionExitStrategyInput) *MeanReversionExitStrategy {
	return &MeanReversionExitStrategy{enabled: input.Enabled}
}

func (strategy *MeanReversionExitStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 || !strategy.enabled {
		return Decision{Action: ActionNone}
	}
	if input.BollMiddle == nil {
		return Decision{Action: ActionNone, Reason: "mean reversion exit: boll_middle unavailable"}
	}
	// Only fire for range-mode entries: entry price was below the mean.
	// Trend entries occur at or above BollMiddle (price above SMA ≈ BollMiddle),
	// so this guard prevents premature exits on trend positions.
	if input.EntryPrice >= *input.BollMiddle {
		return Decision{Action: ActionNone, Reason: "mean reversion exit: entry price at or above bollinger middle"}
	}
	if input.Price < *input.BollMiddle {
		return Decision{Action: ActionNone, Reason: "mean reversion exit: price below bollinger middle"}
	}
	return Decision{
		Action:   ActionSell,
		Reason:   "mean reversion exit: price reached bollinger middle",
		Quantity: input.PositionQuantity,
	}
}
