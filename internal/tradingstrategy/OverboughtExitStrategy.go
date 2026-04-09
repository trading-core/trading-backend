package tradingstrategy

// OverboughtExitStrategy exits when RSI signals the position is overbought,
// unless price is simultaneously making a new N-bar high — which indicates a
// genuine breakout rather than exhaustion. When LookbackHighPrice is zero
// (lookback disabled), the breakout guard is inactive and RSI alone triggers the exit.
//
// When overboughtRSI is zero the strategy is disabled. Missing indicator data
// returns ActionNone rather than silently passing.
type OverboughtExitStrategy struct {
	overboughtRSI float64
}

type NewOverboughtExitStrategyInput struct {
	OverboughtRSI float64 // e.g. 70; 0 disables
}

func NewOverboughtExitStrategy(input NewOverboughtExitStrategyInput) *OverboughtExitStrategy {
	return &OverboughtExitStrategy{overboughtRSI: input.OverboughtRSI}
}

func (strategy *OverboughtExitStrategy) Evaluate(input EvaluateInput) Decision {
	if input.PositionQuantity <= 0 || strategy.overboughtRSI <= 0 {
		return Decision{Action: ActionNone}
	}
	if input.RSI == nil {
		return Decision{Action: ActionNone, Reason: "rsi unavailable"}
	}
	if *input.RSI < strategy.overboughtRSI {
		return Decision{Action: ActionNone, Reason: "rsi not overbought"}
	}
	// Price making a new N-bar high indicates a genuine breakout — hold rather than exit.
	// Only applies when lookback data is available (LookbackBars >= 2).
	if input.LookbackHighPrice > 0 && input.Price > input.LookbackHighPrice {
		return Decision{Action: ActionNone, Reason: "rsi overbought but price breaking out above lookback high"}
	}
	return Decision{Action: ActionSell, Reason: "overbought exit: rsi overbought", Quantity: input.PositionQuantity}
}
