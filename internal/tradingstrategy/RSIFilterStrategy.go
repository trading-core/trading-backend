package tradingstrategy

type RSIFilterStrategy struct {
	minRSI float64
	maxRSI float64
}

type NewRSIFilterStrategyInput struct {
	MinRSI float64
	MaxRSI float64
}

func NewRSIFilterStrategy(input NewRSIFilterStrategyInput) *RSIFilterStrategy {
	return &RSIFilterStrategy{
		minRSI: input.MinRSI,
		maxRSI: input.MaxRSI,
	}
}

func (s *RSIFilterStrategy) Evaluate(input EvaluateInput) Decision {
	if input.RSI == nil {
		return Decision{Action: ActionVeto, Reason: "rsi unavailable"}
	}
	if s.minRSI > 0 && *input.RSI < s.minRSI {
		return Decision{Action: ActionVeto, Reason: "rsi below threshold"}
	}
	if s.maxRSI > 0 && *input.RSI > s.maxRSI {
		return Decision{Action: ActionVeto, Reason: "rsi above threshold"}
	}
	return Decision{Action: ActionNone}
}
