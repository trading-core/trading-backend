package tradingstrategy

// Evaluates strategies in order and returns the first
// non-None decision. ActionVeto still short-circuits immediately.
// This is the right primitive for priority-ordered pipelines where you want
// "first signal that fires wins" rather than majority vote.
type CompositeStrategy struct {
	strategies []Strategy
}

func NewCompositeStrategy(strategies ...Strategy) *CompositeStrategy {
	return &CompositeStrategy{strategies: strategies}
}

func (composite *CompositeStrategy) Evaluate(input EvaluateInput) Decision {
	for _, strategy := range composite.strategies {
		decision := strategy.Evaluate(input)
		if decision.Action == ActionVeto {
			return decision
		}
		if decision.Action != ActionNone {
			return decision
		}
	}
	return Decision{Action: ActionNone, Reason: "no strategy fired"}
}
