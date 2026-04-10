package tradingstrategy

// OverboughtExitStrategy exits when RSI signals the position is overbought AND
// price is at or above the upper Bollinger Band — both conditions must hold.
// The Bollinger Band acts as a confirmation that price has reached a structural
// extreme, not merely a high RSI reading.
//
// An optional lookback guard suppresses the exit when price is simultaneously
// making a new N-bar high, indicating a genuine breakout rather than exhaustion.
// When LookbackHighPrice is zero (lookback disabled), the breakout guard is inactive.
//
// When overboughtRSI is zero the strategy is disabled. Missing indicator data
// (RSI or BollUpper) returns ActionNone with a reason rather than silently passing.
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
	if input.BollUpper == nil {
		return Decision{Action: ActionNone, Reason: "boll_upper unavailable"}
	}
	if input.Price < *input.BollUpper {
		return Decision{Action: ActionNone, Reason: "rsi overbought but price below upper bollinger"}
	}
	return Decision{Action: ActionSell, Reason: "overbought exit: rsi overbought and price at upper bollinger", Quantity: input.PositionQuantity}
}
