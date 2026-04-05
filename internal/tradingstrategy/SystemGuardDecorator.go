package tradingstrategy

type SystemGuardDecorator struct {
	decorated Strategy
}

type NewSystemGuardDecoratorInput struct {
	Decorated Strategy
}

func NewSystemGuardDecorator(input NewSystemGuardDecoratorInput) *SystemGuardDecorator {
	return &SystemGuardDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *SystemGuardDecorator) Evaluate(input EvaluateInput) Decision {
	if input.Price <= 0 {
		return Decision{Action: ActionNone, Reason: "price unavailable"}
	}
	if input.HasOpenOrder {
		return Decision{Action: ActionNone, Reason: "waiting for open order to resolve"}
	}
	return decorator.decorated.Evaluate(input)
}
