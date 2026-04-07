package tradingstrategy

type Parameters struct {
	EntryMode              string  `json:"entry_mode,omitempty"`               // "breakout" or "pullback"
	Timeframe              string  `json:"timeframe,omitempty"`                // candle interval used by data sources and indicators (e.g. "1h", "1d"); not used by strategy logic
	MaxPositionFraction    float64 `json:"max_position_fraction,omitempty"`    // fraction of buying power to deploy per trade (e.g. 0.1 = 10%)
	TakeProfitPct          float64 `json:"take_profit_pct,omitempty"`          // percentage gain above entry price to trigger a profit exit (e.g. 0.02 = 2%)
	StopLossPct            float64 `json:"stop_loss_pct,omitempty"`            // trailing stop-loss percentage below the highest price since entry (e.g. 0.02 = 2%)
	SessionStart           int     `json:"session_start,omitempty"`            // hour 0-23, start of session for new entries (e.g. 10 to start at 10:00)
	SessionEnd             int     `json:"session_end,omitempty"`              // hour 0-23, exclusive end of session for new entries and forced exit (e.g. 15 to exit by 15:00)
	ReentryCooldownMinutes int     `json:"reentry_cooldown_minutes,omitempty"` // minutes before re-entering after a stop-loss exit
	OverboughtRSI          float64 `json:"overbought_rsi,omitempty"`           // RSI level at which to exit a position (e.g. 70); 0 disables
	VolatilityTPMultiplier float64 `json:"volatility_tp_multiplier,omitempty"` // multiplier applied to Bollinger width for a dynamic take-profit; 0 disables
	BreakoutLookbackBars   int     `json:"breakout_lookback_bars,omitempty"`   // bars to look back for breakout high (1 = session high, 5 = 5-bar high)
}

// FromParameters builds a Strategy from Parameters.
// The composite contains: session guard → entry guard → signal strategy.
// Indicator filters (RSI, MACD, Bollinger, SMA) are intentionally excluded;
// compose them manually via NewCompositeStrategy when needed.
func FromParameters(p *Parameters) Strategy {
	signalStrategy := buildSignalStrategy(p)
	composite := NewCompositeStrategy(
		NewSessionGuardStrategy(NewSessionGuardStrategyInput{
			SessionStart:           p.SessionStart,
			SessionEnd:             p.SessionEnd,
			ReentryCooldownMinutes: p.ReentryCooldownMinutes,
			Timeframe:              p.Timeframe,
		}),
		NewEntryGuardStrategy(),
		signalStrategy,
	)
	withSizing := NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
		Decorated:           composite,
		MaxPositionFraction: p.MaxPositionFraction,
		StopLossPct:         p.StopLossPct,
	})
	withExit := NewExitStrategyDecorator(NewExitStrategyDecoratorInput{
		Decorated:              withSizing,
		SessionEnd:             p.SessionEnd,
		TakeProfitPct:          p.TakeProfitPct,
		StopLossPct:            p.StopLossPct,
		VolatilityTPMultiplier: p.VolatilityTPMultiplier,
		OverboughtRSI:          p.OverboughtRSI,
	})
	return NewSystemGuardDecorator(NewSystemGuardDecoratorInput{Decorated: withExit})
}

func buildSignalStrategy(p *Parameters) Strategy {
	switch p.EntryMode {
	case "pullback":
		return new(PullbackStrategy)
	case "breakout":
		return NewBreakoutStrategy(p.BreakoutLookbackBars)
	default:
		panic("unknown strategy entry mode: " + p.EntryMode)
	}
}

var PullbackParameters = Parameters{
	EntryMode:              "pullback",
	Timeframe:              "1d",
	MaxPositionFraction:    0.25,
	TakeProfitPct:          0.08, // 8% — gives hourly positions room to run
	StopLossPct:            0.04, // 4% trailing stop — absorbs normal hourly noise
	SessionStart:           10,   // avoid early open volatility
	SessionEnd:             16,
	ReentryCooldownMinutes: 15,
	VolatilityTPMultiplier: 1.0, // dynamic TP = 100% of Bollinger width when available
}

var BreakoutParameters = Parameters{
	EntryMode:              "breakout",
	Timeframe:              "1h",
	MaxPositionFraction:    0.10,
	TakeProfitPct:          0.05, // 5% target
	StopLossPct:            0.02, // 2% trailing stop
	SessionStart:           9,    // catch early breakouts
	SessionEnd:             16,
	ReentryCooldownMinutes: 5,
	VolatilityTPMultiplier: 1.0,
	BreakoutLookbackBars:   5, // 5-bar high avoids session-noise false breakouts
}
