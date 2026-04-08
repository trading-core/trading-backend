package tradingstrategy

// OverboughtExitStrategy emits a sell when the position is held and multiple
// overbought signals agree: price at or above the upper Bollinger band, RSI above
// the overbought threshold, and MACD crossing below its signal line.
// All three indicator conditions must be met; missing data skips that check.
type OverboughtExitStrategy struct {
	overboughtRSI float64
}

type NewOverboughtExitStrategyInput struct {
	OverboughtRSI float64 // e.g. 70; 0 disables the RSI check
}

func NewOverboughtExitStrategy(input NewOverboughtExitStrategyInput) *OverboughtExitStrategy {
	return &OverboughtExitStrategy{overboughtRSI: input.OverboughtRSI}
}

func (strategy *OverboughtExitStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 {
		return Decision{Action: ActionNone}
	}

	// Bollinger upper band: price must be at or above it.
	if input.BollUpper != nil && input.Price < *input.BollUpper {
		return Decision{Action: ActionNone, Reason: "price below upper bollinger"}
	}

	// RSI: must be above the overbought threshold.
	if strategy.overboughtRSI > 0 && input.RSI != nil && *input.RSI < strategy.overboughtRSI {
		return Decision{Action: ActionNone, Reason: "rsi not overbought"}
	}

	// MACD: must be below or crossing below its signal line.
	if input.MACD != nil && input.MACDSignal != nil && *input.MACD > *input.MACDSignal {
		return Decision{Action: ActionNone, Reason: "macd still above signal"}
	}

	return Decision{Action: ActionSell, Reason: "overbought exit: upper bollinger; rsi overbought; macd below signal", Quantity: input.PositionQuantity}
}
