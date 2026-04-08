package tradingstrategy

// MACDCrossExitStrategy sells when MACD crosses below its signal line while in a
// position, indicating trend momentum has reversed.
//
// To avoid false exits on oversold entries (where MACD may already be below signal
// at the time of entry), the exit only fires once MACD has been confirmed above the
// signal line at some point since the position was opened.
type MACDCrossExitStrategy struct{}

func NewMACDCrossExitStrategy() *MACDCrossExitStrategy {
	return &MACDCrossExitStrategy{}
}

func (s *MACDCrossExitStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 {
		return Decision{Action: ActionNone}
	}
	if input.MACD == nil || input.MACDSignal == nil {
		return Decision{Action: ActionNone, Reason: "macd unavailable"}
	}
	// Only exit if MACD has been above signal at some point since entry.
	// This prevents immediately exiting oversold entries where MACD starts below signal.
	if !input.MACDAboveSinceEntry {
		return Decision{Action: ActionNone, Reason: "macd never above signal since entry"}
	}
	if *input.MACD >= *input.MACDSignal {
		return Decision{Action: ActionNone, Reason: "macd still above signal"}
	}
	return Decision{Action: ActionSell, Reason: "macd crossed below signal", Quantity: input.PositionQuantity}
}
