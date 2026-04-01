package tradingstrategy

type Unknown struct{}

func (Unknown) Type() StrategyType {
	return "unknown"
}

func (Unknown) Evaluate(input EvaluateInput) Decision {
	return Decision{Action: ActionNone, Reason: "unknown strategy type"}
}
