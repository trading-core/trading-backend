package tradingstrategy

// OversoldEntryStrategy emits a buy when not in a position and multiple oversold
// signals agree: price at or below the lower Bollinger band, RSI below the oversold
// threshold, and MACD crossing above its signal line (momentum turning).
// All three indicator conditions must be met; missing data skips that check.
type OversoldEntryStrategy struct {
	oversoldRSI float64
}

type NewOversoldEntryStrategyInput struct {
	OversoldRSI float64 // e.g. 30; 0 disables the RSI check
}

func NewOversoldEntryStrategy(input NewOversoldEntryStrategyInput) *OversoldEntryStrategy {
	return &OversoldEntryStrategy{oversoldRSI: input.OversoldRSI}
}

func (strategy *OversoldEntryStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity > 0 {
		return Decision{Action: ActionNone}
	}

	// Bollinger lower band: price must be at or below it.
	if input.BollLower != nil && input.Price > *input.BollLower {
		return Decision{Action: ActionNone, Reason: "price above lower bollinger"}
	}

	// RSI: must be below the oversold threshold.
	if strategy.oversoldRSI > 0 && input.RSI != nil && *input.RSI > strategy.oversoldRSI {
		return Decision{Action: ActionNone, Reason: "rsi not oversold"}
	}

	// MACD: must be crossing above or already above its signal line.
	if input.MACD != nil && input.MACDSignal != nil && *input.MACD < *input.MACDSignal {
		return Decision{Action: ActionNone, Reason: "macd still below signal"}
	}

	return Decision{Action: ActionBuy, Reason: "oversold entry: lower bollinger; rsi oversold; macd above signal"}
}
