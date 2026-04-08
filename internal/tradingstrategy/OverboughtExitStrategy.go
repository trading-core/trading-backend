package tradingstrategy

// OverboughtExitStrategy emits a sell when the position is held and multiple
// overbought signals agree: price at or above the upper Bollinger band, RSI above
// the overbought threshold, and MACD crossing below its signal line.
//
// Both Bollinger upper band and RSI (when configured) must be present — missing
// indicator data returns ActionNone rather than silently passing the check.
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

	// Bollinger upper band is the primary signal — require it to be present.
	if input.BollUpper == nil {
		return Decision{Action: ActionNone, Reason: "bollinger unavailable"}
	}
	if input.Price < *input.BollUpper {
		return Decision{Action: ActionNone, Reason: "price below upper bollinger"}
	}

	// RSI confirmation — require it when a threshold is configured.
	if strategy.overboughtRSI > 0 {
		if input.RSI == nil {
			return Decision{Action: ActionNone, Reason: "rsi unavailable"}
		}
		if *input.RSI < strategy.overboughtRSI {
			return Decision{Action: ActionNone, Reason: "rsi not overbought"}
		}
	}

	// MACD: must be below or crossing below its signal line.
	if input.MACD != nil && input.MACDSignal != nil && *input.MACD > *input.MACDSignal {
		return Decision{Action: ActionNone, Reason: "macd still above signal"}
	}

	return Decision{Action: ActionSell, Reason: "overbought exit: upper bollinger; rsi overbought; macd below signal", Quantity: input.PositionQuantity}
}
