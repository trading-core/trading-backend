package tradingstrategy

// BreakoutEntryStrategy buys when price makes a new N-bar high, signalling a
// long-term breakout above a sustained resistance level. It is suited for
// capturing trend continuation moves that begin before MACD or SMA have time
// to confirm (e.g. a sudden institutional-driven breakout).
//
// LookbackHighPrice must be populated by the caller with the highest price over
// the prior N bars (excluding the current bar). When it is zero or the position
// is already open, the strategy abstains.
type BreakoutEntryStrategy struct {
	lookbackBars int
}

type NewBreakoutEntryStrategyInput struct {
	LookbackBars int // number of prior bars for the high; must be >= 2 to enable
}

func NewBreakoutEntryStrategy(input NewBreakoutEntryStrategyInput) *BreakoutEntryStrategy {
	return &BreakoutEntryStrategy{
		lookbackBars: input.LookbackBars,
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
	return Decision{Action: ActionBuy, Reason: "breakout entry: price above lookback high"}
}
