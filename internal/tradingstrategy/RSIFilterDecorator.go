package tradingstrategy

type RSIFilterDecorator struct {
	minRSI    float64
	decorated Strategy
}

type NewRSIFilterDecoratorInput struct {
	Decorated Strategy
	MinRSI    float64
}

func NewRSIFilterDecorator(input NewRSIFilterDecoratorInput) *RSIFilterDecorator {
	return &RSIFilterDecorator{
		decorated: input.Decorated,
		minRSI:    input.MinRSI,
	}
}

func (decorator *RSIFilterDecorator) Evaluate(input EvaluateInput) Decision {
	if input.RSI == nil {
		return Decision{Action: ActionNone, Reason: "rsi unavailable"}
	}
	if *input.RSI < decorator.minRSI {
		return Decision{Action: ActionNone, Reason: "rsi below threshold"}
	}
	return decorator.decorated.Evaluate(input)
}
