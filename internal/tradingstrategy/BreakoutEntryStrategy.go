package tradingstrategy

// BreakoutEntryStrategy buys when price makes a new N-bar high, signalling a
// long-term breakout above a sustained resistance level. It is suited for
// capturing trend continuation moves that begin before MACD or SMA have time
// to confirm (e.g. a sudden institutional-driven breakout).
//
// LookbackHighPrice must be populated by the caller with the highest price over
// the prior N bars (excluding the current bar). When it is zero or the position
// is already open, the strategy abstains.
//
// A Bollinger upper-band guard prevents entries when price is already above the
// upper band — a breakout that has run into overextended territory offers a
// poor risk:reward entry. This mirrors the same guard in TrendEntryStrategy.
type BreakoutEntryStrategy struct {
	lookbackBars  int
	overboughtRSI float64
}

type NewBreakoutEntryStrategyInput struct {
	LookbackBars  int     // number of prior bars for the high; must be >= 2 to enable
	OverboughtRSI float64 // RSI threshold above which entry is blocked; 0 disables
}

func NewBreakoutEntryStrategy(input NewBreakoutEntryStrategyInput) *BreakoutEntryStrategy {
	return &BreakoutEntryStrategy{
		lookbackBars:  input.LookbackBars,
		overboughtRSI: input.OverboughtRSI,
	}
}

func (strategy *BreakoutEntryStrategy) Evaluate(input EvaluateInput) Decision {
	if strategy.lookbackBars < 2 {
		return Decision{Action: ActionNone, Reason: "breakout entry disabled"}
	}
	if input.PositionQuantity > 0 {
		return Decision{Action: ActionNone}
	}
	if input.LookbackHighPrice <= 0 {
		return Decision{Action: ActionNone, Reason: "lookback high unavailable"}
	}
	if input.Price <= input.LookbackHighPrice {
		return Decision{Action: ActionNone, Reason: "price not above lookback high"}
	}

	// Reject entries when price is already above the upper Bollinger Band —
	// the breakout has run into overextended territory and the risk:reward is poor.
	if input.BollUpper != nil && input.Price >= *input.BollUpper {
		return Decision{Action: ActionNone, Reason: "breakout entry: price at or above upper bollinger"}
	}

	// Reject if RSI is already overbought at the breakout point — buying into exhaustion.
	if strategy.overboughtRSI > 0 {
		if input.RSI == nil {
			return Decision{Action: ActionNone, Reason: "rsi unavailable"}
		}
		if *input.RSI >= strategy.overboughtRSI {
			return Decision{Action: ActionNone, Reason: "rsi overbought at breakout"}
		}
	}

	return Decision{Action: ActionBuy, Reason: "breakout entry: price above lookback high"}
}
