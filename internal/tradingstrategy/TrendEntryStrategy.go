package tradingstrategy

// TrendEntryStrategy buys when momentum and trend conditions agree:
// MACD above signal and price above SMA.
// All conditions are AND-gated internally. When any condition fails or data is
// missing this strategy returns ActionNone (no opinion), allowing downstream
// strategies in a FirstMatchStrategy pipeline to be evaluated.
type TrendEntryStrategy struct {
	overboughtRSI float64
}

type NewTrendEntryStrategyInput struct {
	OverboughtRSI float64 // RSI threshold above which entry is blocked; 0 disables
}

func NewTrendEntryStrategy(input NewTrendEntryStrategyInput) *TrendEntryStrategy {
	return &TrendEntryStrategy{
		overboughtRSI: input.OverboughtRSI,
	}
}

func (s *TrendEntryStrategy) Evaluate(input EvaluateInput) Decision {
	// MACD must be above signal.
	if input.MACD == nil || input.MACDSignal == nil {
		return Decision{Action: ActionNone, Reason: "macd unavailable"}
	}
	if *input.MACD <= *input.MACDSignal {
		return Decision{Action: ActionNone, Reason: "macd not above signal"}
	}

	// Price must be above SMA.
	if input.SMA == nil {
		return Decision{Action: ActionNone, Reason: "sma unavailable"}
	}
	if input.Price <= *input.SMA {
		return Decision{Action: ActionNone, Reason: "price not above sma"}
	}

	// Reject entries when price is at or above the upper Bollinger band —
	// entering an already-overextended move increases reversal risk.
	if input.BollUpper != nil && input.Price >= *input.BollUpper {
		return Decision{Action: ActionNone, Reason: "price at or above upper bollinger"}
	}

	// Reject entries when RSI is overbought — momentum may be exhausted.
	if s.overboughtRSI > 0 {
		if input.RSI == nil {
			return Decision{Action: ActionNone, Reason: "rsi unavailable"}
		}
		if *input.RSI >= s.overboughtRSI {
			return Decision{Action: ActionNone, Reason: "rsi overbought"}
		}
	}

	return Decision{Action: ActionBuy, Reason: "trend entry: macd above signal; price above sma"}
}
