package tradingstrategy

type IndicatorFilterDecorator struct {
	minRSI                   float64
	requireMACDSignal        bool
	requireBollingerBreakout bool
	minBollingerWidthPct     float64
	requireBollingerSqueeze  bool
	maxBollingerWidthPct     float64
	decorated                Strategy
}

type NewIndicatorFilterDecoratorInput struct {
	Decorated                Strategy
	MinRSI                   float64
	RequireMACDSignal        bool
	RequireBollingerBreakout bool
	MinBollingerWidthPct     float64
	RequireBollingerSqueeze  bool
	MaxBollingerWidthPct     float64
}

func NewIndicatorFilterDecorator(input NewIndicatorFilterDecoratorInput) *IndicatorFilterDecorator {
	return &IndicatorFilterDecorator{
		decorated:                input.Decorated,
		minRSI:                   input.MinRSI,
		requireMACDSignal:        input.RequireMACDSignal,
		requireBollingerBreakout: input.RequireBollingerBreakout,
		minBollingerWidthPct:     input.MinBollingerWidthPct,
		requireBollingerSqueeze:  input.RequireBollingerSqueeze,
		maxBollingerWidthPct:     input.MaxBollingerWidthPct,
	}
}

func (decorator *IndicatorFilterDecorator) Evaluate(input EvaluateInput) Decision {
	if input.RSI == nil {
		return Decision{Action: ActionNone, Reason: "rsi unavailable"}
	}
	if *input.RSI < decorator.minRSI {
		return Decision{Action: ActionNone, Reason: "rsi below threshold"}
	}

	if decorator.requireMACDSignal {
		if input.MACD == nil || input.MACDSignal == nil {
			return Decision{Action: ActionNone, Reason: "macd unavailable"}
		}
		if *input.MACD <= *input.MACDSignal {
			return Decision{Action: ActionNone, Reason: "macd below signal"}
		}
	}

	if decorator.requireBollingerBreakout {
		if input.BollUpper == nil || input.BollMiddle == nil || input.BollLower == nil {
			return Decision{Action: ActionNone, Reason: "bollinger unavailable"}
		}
		if input.Price <= *input.BollUpper {
			return Decision{Action: ActionNone, Reason: "price below upper bollinger"}
		}
		if decorator.minBollingerWidthPct > 0 {
			if input.BollWidthPct == nil {
				return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
			}
			if *input.BollWidthPct < decorator.minBollingerWidthPct {
				return Decision{Action: ActionNone, Reason: "bollinger width too narrow"}
			}
		}
	}

	if decorator.requireBollingerSqueeze {
		if input.BollWidthPct == nil {
			return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
		}
		if *input.BollWidthPct >= decorator.maxBollingerWidthPct {
			return Decision{Action: ActionNone, Reason: "bollinger not in squeeze"}
		}
	}

	return decorator.decorated.Evaluate(input)
}

func (decorator *IndicatorFilterDecorator) Type() StrategyType {
	return decorator.decorated.Type()
}
