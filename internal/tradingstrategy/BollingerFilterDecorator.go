package tradingstrategy

type BollingerFilterDecorator struct {
	requireBreakout bool
	minWidthPct     float64
	maxWidthPct     float64
	decorated       Strategy
}

type NewBollingerFilterDecoratorInput struct {
	Decorated       Strategy
	RequireBreakout bool
	MinWidthPct     float64
	MaxWidthPct     float64
}

func NewBollingerFilterDecorator(input NewBollingerFilterDecoratorInput) *BollingerFilterDecorator {
	return &BollingerFilterDecorator{
		decorated:       input.Decorated,
		requireBreakout: input.RequireBreakout,
		minWidthPct:     input.MinWidthPct,
		maxWidthPct:     input.MaxWidthPct,
	}
}

func (decorator *BollingerFilterDecorator) Evaluate(input EvaluateInput) Decision {
	if decorator.requireBreakout {
		if input.BollUpper == nil || input.BollMiddle == nil || input.BollLower == nil {
			return Decision{Action: ActionNone, Reason: "bollinger unavailable"}
		}
		if input.Price <= *input.BollUpper {
			return Decision{Action: ActionNone, Reason: "price below upper bollinger"}
		}
		if decorator.minWidthPct > 0 {
			if input.BollWidthPct == nil {
				return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
			}
			if *input.BollWidthPct < decorator.minWidthPct {
				return Decision{Action: ActionNone, Reason: "bollinger width too narrow"}
			}
		}
	}

	if decorator.maxWidthPct > 0 {
		if input.BollWidthPct == nil {
			return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
		}
		if *input.BollWidthPct >= decorator.maxWidthPct {
			return Decision{Action: ActionNone, Reason: "bollinger not in squeeze"}
		}
	}

	return decorator.decorated.Evaluate(input)
}
