package tradingstrategy

type Noop struct{}

func (Noop) Type() StrategyType {
	return "noop"
}

func (Noop) Evaluate(input EvaluateInput) Decision {
	return Decision{Action: ActionNone, Reason: "noop strategy"}
}
