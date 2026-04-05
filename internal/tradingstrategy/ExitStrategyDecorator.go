package tradingstrategy

type ExitStrategyDecorator struct {
	sessionEnd             int
	takeProfitPct          float64
	stopLossPct            float64
	volatilityTPMultiplier float64
	decorated              Strategy
}

type NewExitStrategyDecoratorInput struct {
	Decorated              Strategy
	SessionEnd             int     // hour 0-23, exclusive
	TakeProfitPct          float64 // e.g. 0.02 for 2% TP
	StopLossPct            float64 // e.g. 0.01 for 1% trailing stop
	VolatilityTPMultiplier float64 // multiplier for Bollinger width to calculate dynamic TP
}

func NewExitStrategyDecorator(input NewExitStrategyDecoratorInput) *ExitStrategyDecorator {
	return &ExitStrategyDecorator{
		decorated:              input.Decorated,
		sessionEnd:             input.SessionEnd,
		takeProfitPct:          input.TakeProfitPct,
		stopLossPct:            input.StopLossPct,
		volatilityTPMultiplier: input.VolatilityTPMultiplier,
	}
}

func (decorator *ExitStrategyDecorator) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 {
		return decorator.decorated.Evaluate(input)
	}
	hour := input.Now.In(USMarketLocation).Hour()
	if decorator.sessionEnd > 0 && hour >= decorator.sessionEnd {
		return Decision{Action: ActionSell, Reason: "forced end-of-day exit", Quantity: input.PositionQuantity}
	}
	if decorator.takeProfitPct > 0 && input.EntryPrice > 0 {
		effectiveTP := decorator.takeProfitPct
		// Optional volatility-based TP (Bollinger width)
		if decorator.volatilityTPMultiplier > 0 && input.BollWidthPct != nil {
			dynamicTP := *input.BollWidthPct * decorator.volatilityTPMultiplier
			if dynamicTP > effectiveTP {
				effectiveTP = dynamicTP
			}
		}
		if input.Price >= input.EntryPrice*(1+effectiveTP) {
			return Decision{Action: ActionSell, Reason: "take-profit target reached", Quantity: input.PositionQuantity}
		}
	}
	if decorator.stopLossPct > 0 && input.EntryPrice > 0 {
		trailingHigh := input.HighSinceEntry
		if trailingHigh == 0 {
			trailingHigh = input.EntryPrice
		}
		stopLevel := trailingHigh * (1 - decorator.stopLossPct)
		if input.Price <= stopLevel {
			return Decision{Action: ActionSell, Reason: "trailing stop triggered", Quantity: input.PositionQuantity}
		}
	}
	return Decision{Action: ActionNone, Reason: "holding position"}
}
