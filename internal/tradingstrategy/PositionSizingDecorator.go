package tradingstrategy

import "math"

type PositionSizingDecorator struct {
	maxPositionFraction float64
	riskPerTradePct     float64
	stopLossPct         float64
	decorated           Strategy
}

type NewPositionSizingDecoratorInput struct {
	Decorated           Strategy
	MaxPositionFraction float64
	RiskPerTradePct     float64
	StopLossPct         float64
}

func NewPositionSizingDecorator(input NewPositionSizingDecoratorInput) *PositionSizingDecorator {
	return &PositionSizingDecorator{
		decorated:           input.Decorated,
		maxPositionFraction: input.MaxPositionFraction,
		riskPerTradePct:     input.RiskPerTradePct,
		stopLossPct:         input.StopLossPct,
	}
}

func (decorator *PositionSizingDecorator) Evaluate(input EvaluateInput) Decision {
	buyingPower := input.BuyingPower
	if buyingPower <= 0 {
		buyingPower = input.CashBalance
	}
	if buyingPower <= 0 {
		return Decision{Action: ActionNone, Reason: "no buying power available"}
	}

	decision := decorator.decorated.Evaluate(input)
	if decision.Action != ActionBuy {
		return decision
	}

	var qty float64
	if decorator.riskPerTradePct > 0 && decorator.stopLossPct > 0 {
		riskAmount := buyingPower * decorator.riskPerTradePct
		stopDistance := input.Price * decorator.stopLossPct
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
		return Decision{Action: ActionNone, Reason: "insufficient buying power for one share"}
	}
	decision.Quantity = qty
	return decision
}

