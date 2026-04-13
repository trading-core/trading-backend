package tradingstrategy

type Parameters struct {
	Timeframe              string  `json:"timeframe,omitempty"`                // candle interval used by data sources and indicators (e.g. "1h", "1d"); not used by strategy logic
	MaxPositionFraction    float64 `json:"max_position_fraction,omitempty"`    // fraction of buying power to deploy per trade (e.g. 0.1 = 10%)
	ATRMultiplier          float64 `json:"atr_multiplier,omitempty"`           // ATR multiple for trailing stop (e.g. 2.0 = 2×ATR below highSinceEntry); 0 disables
	SessionStart           int     `json:"session_start,omitempty"`            // hour 0-23, start of session for new entries; 0 disables (recommended for 1d)
	SessionEnd             int     `json:"session_end,omitempty"`              // hour 0-23, exclusive end of session for new entries and forced exit; 0 disables
	ReentryCooldownMinutes int     `json:"reentry_cooldown_minutes,omitempty"` // minutes before re-entering after a stop-loss exit
	OverSoldRSI            float64 `json:"oversold_rsi,omitempty"`             // RSI threshold for oversold entry (e.g. 30); 0 disables
	OverboughtRSI          float64 `json:"overbought_rsi,omitempty"`           // RSI threshold for overbought exit (e.g. 70); 0 disables
	LookbackBars           int     `json:"lookback_bars,omitempty"`            // N-bar high used by BreakoutEntryStrategy; 0 disables breakout entry
	RiskPerTradePct        float64 `json:"risk_per_trade_pct,omitempty"`       // fraction of buying power to risk per trade for ATR-based position sizing (e.g. 0.01 = 1%); 0 uses max_position_fraction instead
	ADXThreshold           float64 `json:"adx_threshold,omitempty"`            // ADX value at or above which a trend regime is confirmed (e.g. 20); 0 disables ADX filtering
}

// FromParameters builds a Strategy from Parameters using a priority pipeline:
//
//	SystemGuard (veto on bad market data)
//	└── FirstMatch:
//	      SessionGuard            — veto outside session window (intraday only; disable with zero values)
//	      OverboughtExitStrategy  — exit when RSI overbought + price at upper Bollinger
//	      ATRStopStrategy         — last-resort exit; disabled when atr_multiplier=0
//	      PositionSizingDecorator
//	        └── RegimeSwitchStrategy (EMABasedRegimeDetector):
//	              Uptrend   → FirstMatch(TrendEntry, BreakoutEntry)
//	              Range     → OversoldEntry
//	              Downtrend → NoopStrategy
//	            Regime detection requires FastEMA and SlowEMA in EvaluateInput (computed from
//	            BACKTEST_FAST_EMA_PERIOD / BACKTEST_SLOW_EMA_PERIOD). When either EMA is absent,
//	            the detector defaults to RegimeRange (OversoldEntry only).
func FromParameters(parameters *Parameters) Strategy {
	var strategy Strategy
	strategy = NewRegimeSwitchStrategy(NewRegimeSwitchStrategyInput{
		Detector: NewEMABasedRegimeDetector(NewEMABasedRegimeDetectorInput{
			ADXThreshold: parameters.ADXThreshold,
		}),
		Uptrend: NewCompositeStrategy(
			NewTrendEntryStrategy(NewTrendEntryStrategyInput{
				OverboughtRSI: parameters.OverboughtRSI,
			}),
			NewBreakoutEntryStrategy(NewBreakoutEntryStrategyInput{
				LookbackBars:  parameters.LookbackBars,
				OverboughtRSI: parameters.OverboughtRSI,
			}),
		),
		Range: NewOversoldEntryStrategy(NewOversoldEntryStrategyInput{
			OversoldRSI: parameters.OverSoldRSI,
		}),
		Downtrend: NewNoopStrategy(),
	})
	strategy = NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
		Decorated:           strategy,
		MaxPositionFraction: parameters.MaxPositionFraction,
		ATRMultiplier:       parameters.ATRMultiplier,
		RiskPerTradePct:     parameters.RiskPerTradePct,
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
