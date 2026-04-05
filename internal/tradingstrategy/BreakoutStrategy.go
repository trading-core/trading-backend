package tradingstrategy

import "fmt"

type BreakoutStrategy struct {
	lookbackBars int
}

type NewBreakoutStrategyInput struct {
	LookbackBars int // number of bars to lookback for breakout (1=session high, 5=5-bar high). Default 1.
}

func NewBreakoutStrategy(input NewBreakoutStrategyInput) *BreakoutStrategy {
	return &BreakoutStrategy{
		lookbackBars: input.LookbackBars,
	}
}

// Breakout entry: price breaks above a reference high (session-based or lookback-based).
// For 1-min scalping: use SessionHighPrice (resets daily, tracks intraday range).
// For daily/weekly: use LookbackHighPrice (e.g., 5-bar high, avoids noisy daily resets).
func (strategy *BreakoutStrategy) Evaluate(input EvaluateInput) Decision {
	referenceHigh := input.SessionHighPrice
	if strategy.lookbackBars > 1 && input.LookbackHighPrice > 0 {
		referenceHigh = input.LookbackHighPrice
	}
	if referenceHigh > 0 && input.Price > referenceHigh {
		return Decision{
			Action: ActionBuy,
			Reason: fmt.Sprintf("breakout above %d-bar high", strategy.lookbackBars),
		}
	}
	return Decision{Action: ActionNone, Reason: "no breakout"}
}

