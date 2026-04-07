package tradingstrategy

import "strings"

// CompositeStrategy aggregates multiple strategies via majority vote.
// Any ActionVeto from a child strategy short-circuits and overrides the result.
// In a tie between Buy and Sell, ActionNone is returned.
type CompositeStrategy struct {
	strategies []Strategy
}

func NewCompositeStrategy(strategies ...Strategy) *CompositeStrategy {
	return &CompositeStrategy{strategies: strategies}
}

func (c *CompositeStrategy) Evaluate(input EvaluateInput) Decision {
	var buyDecisions, sellDecisions []Decision

	for _, s := range c.strategies {
		d := s.Evaluate(input)
		switch d.Action {
		case ActionVeto:
			return d
		case ActionBuy:
			buyDecisions = append(buyDecisions, d)
		case ActionSell:
			sellDecisions = append(sellDecisions, d)
		// ActionNone: abstain — guard/filter strategies pass without voting
		}
	}

	var winner []Decision
	var winnerAction Action
	switch {
	case len(buyDecisions) > len(sellDecisions):
		winner, winnerAction = buyDecisions, ActionBuy
	case len(sellDecisions) > len(buyDecisions):
		winner, winnerAction = sellDecisions, ActionSell
	default:
		return Decision{Action: ActionNone, Reason: "no signal"}
	}

	reasons := make([]string, 0, len(winner))
	for _, d := range winner {
		if d.Reason != "" {
			reasons = append(reasons, d.Reason)
		}
	}
	return Decision{Action: winnerAction, Reason: strings.Join(reasons, "; ")}
}
