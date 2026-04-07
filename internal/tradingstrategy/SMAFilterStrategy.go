package tradingstrategy

type SMAFilterStrategy struct{}

func NewSMAFilterStrategy() *SMAFilterStrategy {
	return &SMAFilterStrategy{}
}

func (s *SMAFilterStrategy) Evaluate(input EvaluateInput) Decision {
	if input.SMA == nil {
		return Decision{Action: ActionVeto, Reason: "sma unavailable"}
	}
	if input.Price <= *input.SMA {
		return Decision{Action: ActionVeto, Reason: "price below sma"}
	}
	return Decision{Action: ActionNone}
}
