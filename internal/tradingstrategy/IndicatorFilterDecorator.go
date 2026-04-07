package tradingstrategy

type SMAFilterDecorator struct {
	decorated Strategy
}

type NewSMAFilterDecoratorInput struct {
	Decorated Strategy
}

func NewSMAFilterDecorator(input NewSMAFilterDecoratorInput) *SMAFilterDecorator {
	return &SMAFilterDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *SMAFilterDecorator) Evaluate(input EvaluateInput) Decision {
	if input.SMA == nil {
		return Decision{Action: ActionNone, Reason: "sma unavailable"}
	}
	if input.Price <= *input.SMA {
		return Decision{Action: ActionNone, Reason: "price below sma"}
	}
	return decorator.decorated.Evaluate(input)
}
