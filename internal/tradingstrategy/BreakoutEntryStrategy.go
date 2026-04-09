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
// An optional minimum Bollinger band width filter (MinBollingerWidthPct) can be
// used to require that the breakout occurs from a squeeze, avoiding entries
// during already-expanded, noisy conditions.
type BreakoutEntryStrategy struct {
	lookbackBars         int
	minBollingerWidthPct float64
}

type NewBreakoutEntryStrategyInput struct {
	LookbackBars         int     // number of prior bars for the high; must be >= 2 to enable
	MinBollingerWidthPct float64 // minimum band width (% of middle) required; 0 disables
}

func NewBreakoutEntryStrategy(input NewBreakoutEntryStrategyInput) *BreakoutEntryStrategy {
	return &BreakoutEntryStrategy{
		lookbackBars:         input.LookbackBars,
		minBollingerWidthPct: input.MinBollingerWidthPct,
	}
}

func (s *BreakoutEntryStrategy) Evaluate(input EvaluateInput) Decision {
	if s.lookbackBars < 2 {
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
	if s.minBollingerWidthPct > 0 {
		if input.BollWidthPct == nil {
			return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
		}
		if *input.BollWidthPct < s.minBollingerWidthPct {
			return Decision{Action: ActionNone, Reason: "bollinger width too narrow"}
		}
	}
	return Decision{Action: ActionBuy, Reason: "breakout entry: price above lookback high"}
}
