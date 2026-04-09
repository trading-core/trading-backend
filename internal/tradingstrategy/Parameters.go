package tradingstrategy

type Parameters struct {
	Timeframe                    string  `json:"timeframe,omitempty"`                        // candle interval used by data sources and indicators (e.g. "1h", "1d"); not used by strategy logic
	MaxPositionFraction          float64 `json:"max_position_fraction,omitempty"`            // fraction of buying power to deploy per trade (e.g. 0.1 = 10%)
	ATRMultiplier                float64 `json:"atr_multiplier,omitempty"`                   // ATR multiple for trailing stop (e.g. 2.0 = 2×ATR below highSinceEntry); 0 disables
	SessionStart                 int     `json:"session_start,omitempty"`                    // hour 0-23, start of session for new entries; 0 disables (recommended for 1d)
	SessionEnd                   int     `json:"session_end,omitempty"`                      // hour 0-23, exclusive end of session for new entries and forced exit; 0 disables
	ReentryCooldownMinutes       int     `json:"reentry_cooldown_minutes,omitempty"`         // minutes before re-entering after a stop-loss exit
	OverSoldRSI                  float64 `json:"oversold_rsi,omitempty"`                     // RSI threshold for oversold entry (e.g. 30); 0 disables
	OverboughtRSI                float64 `json:"overbought_rsi,omitempty"`                   // RSI threshold for overbought exit (e.g. 70); 0 disables
	LookbackBars                 int     `json:"lookback_bars,omitempty"`                    // N-bar high used by BreakoutEntryStrategy; 0 disables breakout entry
}

// FromParameters builds a Strategy from Parameters using a priority pipeline:
//
//	SystemGuard (veto on bad market data)
//	└── FirstMatch:
//	      SessionGuard            — veto outside session window (intraday only; disable with zero values)
//	      OverboughtExitStrategy  — exit when RSI overbought + price at upper Bollinger
//	      ATRStopStrategy    — last-resort exit; disabled when atr_multiplier=0
//	      PositionSizingDecorator
//	        └── FirstMatch (entry signals, evaluated when flat):
//	              BreakoutEntryStrategy — N-bar high breakout; disabled when lookback_bars=0
//	              TrendEntryStrategy    — MACD + SMA momentum entry
//	              OversoldEntryStrategy — RSI + lower Bollinger mean-reversion entry
func FromParameters(parameters *Parameters) Strategy {
	var strategy Strategy
	strategy = NewCompositeStrategy(
		NewBreakoutEntryStrategy(NewBreakoutEntryStrategyInput{
			LookbackBars: parameters.LookbackBars,
		}),
		NewTrendEntryStrategy(NewTrendEntryStrategyInput{
			OverboughtRSI: parameters.OverboughtRSI,
		}),
		NewOversoldEntryStrategy(NewOversoldEntryStrategyInput{
			OversoldRSI: parameters.OverSoldRSI,
		}),
	)
	strategy = NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
		Decorated:           strategy,
		MaxPositionFraction: parameters.MaxPositionFraction,
		ATRMultiplier:       parameters.ATRMultiplier,
	})
	strategy = NewCompositeStrategy(
		NewSessionGuardStrategy(NewSessionGuardStrategyInput{
			SessionStart:           parameters.SessionStart,
			SessionEnd:             parameters.SessionEnd,
			ReentryCooldownMinutes: parameters.ReentryCooldownMinutes,
			Timeframe:              parameters.Timeframe,
		}),
		NewOverboughtExitStrategy(NewOverboughtExitStrategyInput{
			OverboughtRSI: parameters.OverboughtRSI,
		}),
		NewATRStopStrategy(NewATRStopStrategyInput{
			ATRMultiplier: parameters.ATRMultiplier,
		}),
		strategy,
	)
	return NewSystemGuardDecorator(NewSystemGuardDecoratorInput{Decorated: strategy})
}
