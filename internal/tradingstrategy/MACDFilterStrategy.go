package tradingstrategy

type MACDFilterStrategy struct{}

func NewMACDFilterStrategy() *MACDFilterStrategy {
	return &MACDFilterStrategy{}
}

func (s *MACDFilterStrategy) Evaluate(input EvaluateInput) Decision {
	if input.MACD == nil || input.MACDSignal == nil {
		return Decision{Action: ActionVeto, Reason: "macd unavailable"}
	}
	if *input.MACD <= *input.MACDSignal {
		return Decision{Action: ActionVeto, Reason: "macd below signal"}
	}
	return Decision{Action: ActionNone}
}
