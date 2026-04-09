package tradingstrategy

import (
	"math"
)

type PositionSizingDecorator struct {
	maxPositionFraction float64
	riskPerTradePct     float64
	atrMultiplier       float64
	decorated           Strategy
}

type NewPositionSizingDecoratorInput struct {
	Decorated           Strategy
	MaxPositionFraction float64
	RiskPerTradePct     float64
	ATRMultiplier       float64
}

func NewPositionSizingDecorator(input NewPositionSizingDecoratorInput) *PositionSizingDecorator {
	return &PositionSizingDecorator{
		decorated:           input.Decorated,
		maxPositionFraction: input.MaxPositionFraction,
		riskPerTradePct:     input.RiskPerTradePct,
		atrMultiplier:       input.ATRMultiplier,
	}
}

func (decorator *PositionSizingDecorator) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity > 0 {
		return Decision{Action: ActionNone, Reason: "already in position"}
	}
	buyingPower := input.BuyingPower
	if buyingPower <= 0 {
		buyingPower = input.CashBalance
	}
	if buyingPower <= 0 {
		return Decision{Action: ActionVeto, Reason: "no buying power available"}
	}

	decision := decorator.decorated.Evaluate(input)
	if decision.Action != ActionBuy {
		return decision
	}

	var qty float64
	if decorator.riskPerTradePct > 0 && decorator.atrMultiplier > 0 && input.ATR != nil {
		riskAmount := buyingPower * decorator.riskPerTradePct
		stopDistance := *input.ATR * decorator.atrMultiplier
		qty = math.Floor(riskAmount / stopDistance)
		maxQty := math.Floor(buyingPower * decorator.maxPositionFraction / input.Price)
		if qty > maxQty {
			qty = maxQty
		}
	} else {
		maxCapital := buyingPower * decorator.maxPositionFraction
		qty = math.Floor(maxCapital / input.Price)
	}
	if qty < 1 {
		return Decision{Action: ActionVeto, Reason: "insufficient buying power for one share"}
	}
	decision.Quantity = qty
	return decision
}
