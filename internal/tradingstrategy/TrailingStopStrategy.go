package tradingstrategy

type TrailingStopStrategy struct {
	stopLossPct float64
}

type NewTrailingStopStrategyInput struct {
	StopLossPct float64 // e.g. 0.01 for 1% trailing stop
}

func NewTrailingStopStrategy(input NewTrailingStopStrategyInput) *TrailingStopStrategy {
	return &TrailingStopStrategy{stopLossPct: input.StopLossPct}
}

func (strategy *TrailingStopStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity > 0 && strategy.stopLossPct > 0 && input.EntryPrice > 0 {
		trailingHigh := input.HighSinceEntry
		if trailingHigh == 0 {
			trailingHigh = input.EntryPrice
		}
		stopLevel := trailingHigh * (1 - strategy.stopLossPct)
		if input.Price <= stopLevel {
			return Decision{Action: ActionSell, Reason: "trailing stop triggered", Quantity: input.PositionQuantity}
		}
	}
	return Decision{Action: ActionNone}
}
