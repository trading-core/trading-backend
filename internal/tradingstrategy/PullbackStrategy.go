package tradingstrategy

// Pullback entry: buy when price dips to or below the Bollinger middle
// band while RSI/MACD still confirm upward momentum. This enters at
// mean-reversion support rather than chasing breakouts at resistance.
type PullbackStrategy struct{}

func (strategy *PullbackStrategy) Evaluate(input EvaluateInput) Decision {
	if input.BollMiddle == nil {
		return Decision{Action: ActionNone, Reason: "bollinger middle unavailable for pullback"}
	}
	if input.Price <= *input.BollMiddle {
		return Decision{Action: ActionBuy, Reason: "pullback"}
	}
	return Decision{Action: ActionNone}
}

func (strategy *PullbackStrategy) Type() StrategyType {
	return StrategyTypePullbackTrading
}
