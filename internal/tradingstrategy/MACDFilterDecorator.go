package tradingstrategy

type MACDFilterDecorator struct {
	decorated Strategy
}

type NewMACDFilterDecoratorInput struct {
	Decorated Strategy
}

func NewMACDFilterDecorator(input NewMACDFilterDecoratorInput) *MACDFilterDecorator {
	return &MACDFilterDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *MACDFilterDecorator) Evaluate(input EvaluateInput) Decision {
	if input.MACD == nil || input.MACDSignal == nil {
		return Decision{Action: ActionNone, Reason: "macd unavailable"}
	}
	if *input.MACD <= *input.MACDSignal {
		return Decision{Action: ActionNone, Reason: "macd below signal"}
	}
	return decorator.decorated.Evaluate(input)
}
