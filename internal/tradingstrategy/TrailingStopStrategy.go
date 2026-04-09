package tradingstrategy

// TrailingStopStrategy exits a position when price falls a configured percentage
// below the highest price since entry. Optional confirmations prevent the stop from
// firing on brief dips when indicators still suggest the position is healthy:
//
//   - StopConfirmBelowBollMiddle: only exit if price is also below the Bollinger
//     middle band, indicating the position has fallen into the lower half of the range.
//   - StopConfirmRSIBelow: only exit if RSI is also below this threshold, indicating
//     momentum has genuinely weakened. 0 disables.
//
// When a confirmation is enabled but its indicator data is missing, the stop does
// not fire — missing data is treated as "not confirmed".
type TrailingStopStrategy struct {
	stopLossPct               float64
	stopConfirmBelowBollMiddle bool
	stopConfirmRSIBelow        float64
}

type NewTrailingStopStrategyInput struct {
	StopLossPct               float64 // e.g. 0.05 for 5% trailing stop; 0 disables
	StopConfirmBelowBollMiddle bool    // require price below Bollinger middle before stopping
	StopConfirmRSIBelow        float64 // require RSI below this threshold before stopping; 0 disables
}

func NewTrailingStopStrategy(input NewTrailingStopStrategyInput) *TrailingStopStrategy {
	return &TrailingStopStrategy{
		stopLossPct:               input.StopLossPct,
		stopConfirmBelowBollMiddle: input.StopConfirmBelowBollMiddle,
		stopConfirmRSIBelow:        input.StopConfirmRSIBelow,
	}
}

func (strategy *TrailingStopStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 || strategy.stopLossPct <= 0 || input.EntryPrice <= 0 {
		return Decision{Action: ActionNone}
	}

	trailingHigh := input.HighSinceEntry
	if trailingHigh == 0 {
		trailingHigh = input.EntryPrice
	}
	if input.Price > trailingHigh*(1-strategy.stopLossPct) {
		return Decision{Action: ActionNone}
	}

	// Stop level breached — check confirmations before exiting.
	if strategy.stopConfirmBelowBollMiddle {
		if input.BollMiddle == nil || input.Price >= *input.BollMiddle {
			return Decision{Action: ActionNone, Reason: "stop level reached but price above bollinger middle"}
		}
	}
	if strategy.stopConfirmRSIBelow > 0 {
		if input.RSI == nil || *input.RSI >= strategy.stopConfirmRSIBelow {
			return Decision{Action: ActionNone, Reason: "stop level reached but rsi not confirming weakness"}
		}
	}

	return Decision{Action: ActionSell, Reason: "trailing stop triggered", Quantity: input.PositionQuantity}
}
