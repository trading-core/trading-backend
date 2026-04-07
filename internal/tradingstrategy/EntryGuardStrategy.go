package tradingstrategy

type EntryGuardStrategy struct{}

func NewEntryGuardStrategy() *EntryGuardStrategy {
	return &EntryGuardStrategy{}
}

func (strategy *EntryGuardStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity > 0 {
		return Decision{Action: ActionVeto, Reason: "already in position"}
	}
	return Decision{Action: ActionNone}
}
