package tradingstrategy

type TakeProfitStrategy struct {
	takeProfitPct          float64
	volatilityTPMultiplier float64
}

type NewTakeProfitStrategyInput struct {
	TakeProfitPct          float64 // e.g. 0.02 for 2% TP
	VolatilityTPMultiplier float64 // multiplier for Bollinger width to calculate dynamic TP
}

func NewTakeProfitStrategy(input NewTakeProfitStrategyInput) *TakeProfitStrategy {
	return &TakeProfitStrategy{
		takeProfitPct:          input.TakeProfitPct,
		volatilityTPMultiplier: input.VolatilityTPMultiplier,
	}
}

func (strategy *TakeProfitStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity > 0 && strategy.takeProfitPct > 0 && input.EntryPrice > 0 {
		effectiveTP := strategy.takeProfitPct
		if strategy.volatilityTPMultiplier > 0 && input.BollWidthPct != nil {
			dynamicTP := *input.BollWidthPct * strategy.volatilityTPMultiplier
			if dynamicTP > effectiveTP {
				effectiveTP = dynamicTP
			}
		}
		if input.Price >= input.EntryPrice*(1+effectiveTP) {
			return Decision{Action: ActionSell, Reason: "take-profit target reached", Quantity: input.PositionQuantity}
		}
	}
	return Decision{Action: ActionNone}
}
