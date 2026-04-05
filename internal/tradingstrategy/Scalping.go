package tradingstrategy

// Scalping strategy holds parameters.
// MaxPositionFraction is the fraction of buying power to deploy per trade (e.g. 0.1 = 10%).
// TakeProfitPct is the percentage gain above entry price to trigger a profit exit (e.g. 0.005 = 0.5%).
// StopLossPct is the trailing stop-loss percentage below the highest price since entry (e.g. 0.02 = 2%).
// SessionStart and SessionEnd define the window (hour in exchange local time) during which new entries are allowed.
// Positions are force-closed when the hour reaches SessionEnd.

type Scalping struct {
	EntryMode                string // "breakout" or "pullback"
	MaxPositionFraction      float64
	TakeProfitPct            float64
	StopLossPct              float64
	SessionStart             int // hour 0-23
	SessionEnd               int // hour 0-23, exclusive
	MinRSI                   float64
	RequireMACDSignal        bool
	RequireBollingerBreakout bool
	MinBollingerWidthPct     float64
	RequireBollingerSqueeze  bool
	MaxBollingerWidthPct     float64
	ReentryCooldownMinutes   int
	UseVolatilityTP          bool
	VolatilityTPMultiplier   float64
	RiskPerTradePct          float64
	BreakoutLookbackBars     int // number of bars to lookback for breakout (1=session high, 5=5-bar high)
}

func NewScalping() *Scalping {
	return &Scalping{
		EntryMode:                "pullback",
		MaxPositionFraction:      0.1,
		TakeProfitPct:            0.005,
		StopLossPct:              0.02,
		SessionStart:             10,
		SessionEnd:               15,
		MinRSI:                   40,
		RequireMACDSignal:        true,
		RequireBollingerBreakout: false,
		MinBollingerWidthPct:     0,
		RequireBollingerSqueeze:  false,
		MaxBollingerWidthPct:     0.02,
		ReentryCooldownMinutes:   5,
		UseVolatilityTP:          false,
		VolatilityTPMultiplier:   0.5,
		RiskPerTradePct:          0,
		BreakoutLookbackBars:     1,
	}
}

func (strategy *Scalping) Evaluate(input EvaluateInput) Decision {
	if input.HasOpenOrder {
		return Decision{Action: ActionNone, Reason: "waiting for open order to resolve"}
	}
	if input.Price <= 0 {
		return Decision{Action: ActionNone, Reason: "price unavailable"}
	}

	decision := strategy.newDecisionEngine().Evaluate(input)
	if input.PositionQuantity == 0 && decision.Action == ActionNone && decision.Reason == "" {
		return Decision{Action: ActionNone, Reason: "no entry signal"}
	}
	if input.PositionQuantity == 0 && decision.Action == ActionBuy {
		decision.Reason = "entry signal: " + strategy.EntryMode
	}
	return decision
}

func (strategy *Scalping) newDecisionEngine() Strategy {
	var tradingStrategy Strategy
	tradingStrategy = strategy.newEntrySignalStrategy()
	tradingStrategy = NewEntryStrategyDecorator(NewEntryStrategyDecoratorInput{
		Decorated: tradingStrategy,
	})
	tradingStrategy = NewIndicatorFilterDecorator(NewIndicatorFilterDecoratorInput{
		Decorated:                tradingStrategy,
		MinRSI:                   strategy.MinRSI,
		RequireMACDSignal:        strategy.RequireMACDSignal,
		RequireBollingerBreakout: strategy.RequireBollingerBreakout,
		MinBollingerWidthPct:     strategy.MinBollingerWidthPct,
		RequireBollingerSqueeze:  strategy.RequireBollingerSqueeze,
		MaxBollingerWidthPct:     strategy.MaxBollingerWidthPct,
	})
	tradingStrategy = NewSessionGuardDecorator(NewSessionGuardDecoratorInput{
		Decorated:              tradingStrategy,
		SessionStart:           strategy.SessionStart,
		SessionEnd:             strategy.SessionEnd,
		ReentryCooldownMinutes: strategy.ReentryCooldownMinutes,
	})
	tradingStrategy = NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
		Decorated:           tradingStrategy,
		MaxPositionFraction: strategy.MaxPositionFraction,
		RiskPerTradePct:     strategy.RiskPerTradePct,
		StopLossPct:         strategy.StopLossPct,
	})
	tradingStrategy = NewExitStrategyDecorator(NewExitStrategyDecoratorInput{
		Decorated:              tradingStrategy,
		SessionEnd:             strategy.SessionEnd,
		TakeProfitPct:          strategy.TakeProfitPct,
		StopLossPct:            strategy.StopLossPct,
		UseVolatilityTP:        strategy.UseVolatilityTP,
		VolatilityTPMultiplier: strategy.VolatilityTPMultiplier,
	})
	return tradingStrategy
}

func (strategy *Scalping) newEntrySignalStrategy() Strategy {
	switch strategy.EntryMode {
	case "pullback":
		return &PullbackStrategy{}
	default:
		return NewBreakoutStrategy(NewBreakoutStrategyInput{LookbackBars: strategy.BreakoutLookbackBars})
	}
}
