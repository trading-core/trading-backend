package tradingstrategy

type BollingerFilterStrategy struct {
	requireBreakout bool
	minWidthPct     float64
	maxWidthPct     float64
}

type NewBollingerFilterStrategyInput struct {
	RequireBreakout bool
	MinWidthPct     float64
	MaxWidthPct     float64
}

func NewBollingerFilterStrategy(input NewBollingerFilterStrategyInput) *BollingerFilterStrategy {
	return &BollingerFilterStrategy{
		requireBreakout: input.RequireBreakout,
		minWidthPct:     input.MinWidthPct,
		maxWidthPct:     input.MaxWidthPct,
	}
}

func (s *BollingerFilterStrategy) Evaluate(input EvaluateInput) Decision {
	if s.requireBreakout {
		if input.BollUpper == nil || input.BollMiddle == nil || input.BollLower == nil {
			return Decision{Action: ActionVeto, Reason: "bollinger unavailable"}
		}
		if input.Price <= *input.BollUpper {
			return Decision{Action: ActionVeto, Reason: "price below upper bollinger"}
		}
		if s.minWidthPct > 0 {
			if input.BollWidthPct == nil {
				return Decision{Action: ActionVeto, Reason: "bollinger width unavailable"}
			}
			if *input.BollWidthPct < s.minWidthPct {
				return Decision{Action: ActionVeto, Reason: "bollinger width too narrow"}
			}
		}
	}

	if s.maxWidthPct > 0 {
		if input.BollWidthPct == nil {
			return Decision{Action: ActionVeto, Reason: "bollinger width unavailable"}
		}
		if *input.BollWidthPct >= s.maxWidthPct {
			return Decision{Action: ActionVeto, Reason: "bollinger not in squeeze"}
		}
	}

	return Decision{Action: ActionNone}
}
