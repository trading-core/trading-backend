package tradingstrategy

// OversoldEntryStrategy emits a buy when not in a position and price is at or
// below the lower Bollinger band with RSI confirming oversold conditions.
// MACD is intentionally excluded: during a deep selloff MACD is almost always
// below its signal line, so requiring a crossover would defeat mean-reversion entries.
//
// Both Bollinger lower band and RSI (when configured) must be present — missing
// indicator data returns ActionNone rather than silently passing the check.
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

	// Bollinger lower band is the primary signal — require it to be present.
	if input.BollLower == nil {
		return Decision{Action: ActionNone, Reason: "bollinger unavailable"}
	}
	if input.Price > *input.BollLower {
		return Decision{Action: ActionNone, Reason: "price above lower bollinger"}
	}

	// RSI confirmation — require it when a threshold is configured.
	if strategy.oversoldRSI > 0 {
		if input.RSI == nil {
			return Decision{Action: ActionNone, Reason: "rsi unavailable"}
		}
		if *input.RSI > strategy.oversoldRSI {
			return Decision{Action: ActionNone, Reason: "rsi not oversold"}
		}
	}

	if strategy.oversoldRSI > 0 {
		return Decision{Action: ActionBuy, Reason: "oversold entry: lower bollinger; rsi oversold"}
	}
	return Decision{Action: ActionBuy, Reason: "oversold entry: lower bollinger"}
}
