package tradingstrategy

type EntryStrategyDecorator struct {
	decorated Strategy
}

type NewEntryStrategyDecoratorInput struct {
	Decorated Strategy
}

func NewEntryStrategyDecorator(input NewEntryStrategyDecoratorInput) *EntryStrategyDecorator {
	return &EntryStrategyDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *EntryStrategyDecorator) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity == 0 {
		return decorator.decorated.Evaluate(input)
	}
	return Decision{Action: ActionNone, Reason: "already in position"}
}

func (decorator *EntryStrategyDecorator) Type() StrategyType {
	return decorator.decorated.Type()
}
