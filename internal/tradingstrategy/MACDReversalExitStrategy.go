package tradingstrategy

// MACDReversalExitStrategy exits a trend position when MACD falls below its
// signal line while price is above the entry price. This is the symmetric
// counterpart to TrendEntryStrategy, which requires MACD above signal to enter.
//
// The price-above-entry guard prevents the strategy from firing immediately
// after a range-mode (OversoldEntry) trade where MACD is already bearish at
// entry — the ATRStopStrategy handles downside protection in that scenario.
//
// Missing MACD or MACDSignal data causes the strategy to abstain.
// When Enabled is false the strategy is disabled.
type MACDReversalExitStrategy struct {
	enabled bool
}

type NewMACDReversalExitStrategyInput struct {
	Enabled bool // true to enable; false disables
}

func NewMACDReversalExitStrategy(input NewMACDReversalExitStrategyInput) *MACDReversalExitStrategy {
	return &MACDReversalExitStrategy{enabled: input.Enabled}
}

func (strategy *MACDReversalExitStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 || !strategy.enabled {
		return Decision{Action: ActionNone}
	}
	if input.MACD == nil || input.MACDSignal == nil {
		return Decision{Action: ActionNone, Reason: "macd reversal exit: macd unavailable"}
	}
	// Only exit above entry price — below entry the ATR stop handles protection.
	if input.Price <= input.EntryPrice {
		return Decision{Action: ActionNone, Reason: "macd reversal exit: price at or below entry price"}
	}
	if *input.MACD >= *input.MACDSignal {
		return Decision{Action: ActionNone, Reason: "macd reversal exit: macd above signal"}
	}
	return Decision{
		Action:   ActionSell,
		Reason:   "macd reversal exit: macd crossed below signal above entry price",
		Quantity: input.PositionQuantity,
	}
}
