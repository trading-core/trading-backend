package tradingstrategy

// TrendEntryStrategy buys when momentum, trend, and Bollinger conditions all agree:
// MACD above signal, price above SMA, and price below the upper Bollinger band.
// Optional Bollinger width thresholds filter out low-volatility squeezes or
// excessive band expansion.
//
// All conditions are AND-gated internally. When any condition fails or data is
// missing this strategy returns ActionNone (no opinion), allowing downstream
// strategies in a FirstMatchStrategy pipeline to be evaluated.
type TrendEntryStrategy struct {
	minBollingerWidthPct float64
	maxBollingerWidthPct float64
}

type NewTrendEntryStrategyInput struct {
	MinBollingerWidthPct float64 // minimum band width required; 0 disables
	MaxBollingerWidthPct float64 // maximum band width (squeeze filter); 0 disables
}

func NewTrendEntryStrategy(input NewTrendEntryStrategyInput) *TrendEntryStrategy {
	return &TrendEntryStrategy{
		minBollingerWidthPct: input.MinBollingerWidthPct,
		maxBollingerWidthPct: input.MaxBollingerWidthPct,
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

	// Price must be below the upper Bollinger band (not overbought).
	if input.BollUpper != nil && input.Price >= *input.BollUpper {
		return Decision{Action: ActionNone, Reason: "price at or above upper bollinger"}
	}

	// Optional: band must be wide enough (avoids low-volatility false signals).
	if s.minBollingerWidthPct > 0 {
		if input.BollWidthPct == nil {
			return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
		}
		if *input.BollWidthPct < s.minBollingerWidthPct {
			return Decision{Action: ActionNone, Reason: "bollinger width too narrow"}
		}
	}

	// Optional: band must not be too wide (squeeze detection).
	if s.maxBollingerWidthPct > 0 {
		if input.BollWidthPct == nil {
			return Decision{Action: ActionNone, Reason: "bollinger width unavailable"}
		}
		if *input.BollWidthPct >= s.maxBollingerWidthPct {
			return Decision{Action: ActionNone, Reason: "bollinger not in squeeze"}
		}
	}

	return Decision{Action: ActionBuy, Reason: "trend entry: macd above signal; price above sma; bollinger conditions met"}
}
