package tradingstrategy

type Parameters struct {
	EntryMode                string  `json:"entry_mode,omitempty"`                 // "breakout" or "pullback"
	MaxPositionFraction      float64 `json:"max_position_fraction,omitempty"`      // fraction of buying power to deploy per trade (e.g. 0.1 = 10%)
	TakeProfitPct            float64 `json:"take_profit_pct,omitempty"`            // percentage gain above entry price to trigger a profit exit (e.g. 0.005 = 0.5%)
	StopLossPct              float64 `json:"stop_loss_pct,omitempty"`              // trailing stop-loss percentage below the highest price since entry (e.g. 0.02 = 2%)
	SessionStart             int     `json:"session_start,omitempty"`              // hour 0-23, start of session for new entries (e.g. 10 to start at 10:00)
	SessionEnd               int     `json:"session_end,omitempty"`                // hour 0-23, exclusive end of session for new entries and forced exit (e.g. 15 to exit by 15:00)
	MinRSI                   float64 `json:"min_rsi,omitempty"`                    // minimum RSI required to enter (e.g. 40)
	RequireMACDSignal        bool    `json:"require_macd_signal,omitempty"`        // if true, require positive MACD signal to enter
	RequireBollingerBreakout bool    `json:"require_bollinger_breakout,omitempty"` // if true, require price above upper Bollinger Band to enter
	MinBollingerWidthPct     float64 `json:"min_bollinger_width_pct,omitempty"`    // minimum Bollinger Band width as percentage to enter (e.g. 0.01 for 1%)
	MaxBollingerWidthPct     float64 `json:"max_bollinger_width_pct,omitempty"`    // maximum Bollinger Band width as percentage to enter (e.g. 0.02 for 2%)
	ReentryCooldownMinutes   int     `json:"reentry_cooldown_minutes,omitempty"`   // cooldown period in minutes before re-entering after an exit
	VolatilityTPMultiplier   float64 `json:"volatility_tp_multiplier,omitempty"`   // multiplier for Bollinger width to calculate dynamic TP (e.g. 0.5 for 50% of Bollinger width)
	BreakoutLookbackBars     int     `json:"breakout_lookback_bars,omitempty"`     // number of bars to lookback for breakout (1=session high, 5=5-bar high). Default 1.
	RequirePriceAboveSMA     bool    `json:"require_price_above_sma,omitempty"`    // if true, require price above the SMA to enter (trend filter)
	Timeframe                string  `json:"timeframe,omitempty"`                  // "1m", "5m", "1h", "1d", etc.
}

func FromParameters(parameters *Parameters) Strategy {
	var tradingStrategy Strategy
	switch parameters.EntryMode {
	case "pullback":
		tradingStrategy = new(PullbackStrategy)
	case "breakout":
		tradingStrategy = NewBreakoutStrategy(parameters.BreakoutLookbackBars)
	default:
		panic("unknown strategy entry mode " + parameters.EntryMode)
	}
	tradingStrategy = NewIndicatorFilterDecorator(NewIndicatorFilterDecoratorInput{
		Decorated:                tradingStrategy,
		MinRSI:                   parameters.MinRSI,
		RequireMACDSignal:        parameters.RequireMACDSignal,
		RequireBollingerBreakout: parameters.RequireBollingerBreakout,
		MinBollingerWidthPct:     parameters.MinBollingerWidthPct,
		MaxBollingerWidthPct:     parameters.MaxBollingerWidthPct,
		RequirePriceAboveSMA:     parameters.RequirePriceAboveSMA,
	})
	tradingStrategy = NewEntryStrategyDecorator(NewEntryStrategyDecoratorInput{
		Decorated: tradingStrategy,
	})
	tradingStrategy = NewSessionGuardDecorator(NewSessionGuardDecoratorInput{
		Decorated:              tradingStrategy,
		SessionStart:           parameters.SessionStart,
		SessionEnd:             parameters.SessionEnd,
		ReentryCooldownMinutes: parameters.ReentryCooldownMinutes,
	})
	tradingStrategy = NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
		Decorated:           tradingStrategy,
		MaxPositionFraction: parameters.MaxPositionFraction,
		StopLossPct:         parameters.StopLossPct,
	})
	tradingStrategy = NewExitStrategyDecorator(NewExitStrategyDecoratorInput{
		Decorated:              tradingStrategy,
		SessionEnd:             parameters.SessionEnd,
		TakeProfitPct:          parameters.TakeProfitPct,
		StopLossPct:            parameters.StopLossPct,
		VolatilityTPMultiplier: parameters.VolatilityTPMultiplier,
	})
	tradingStrategy = NewSystemGuardDecorator(NewSystemGuardDecoratorInput{
		Decorated: tradingStrategy,
	})
	return tradingStrategy
}

var PullbackParameters = Parameters{
	EntryMode:                "pullback",
	Timeframe:                "1h",
	MaxPositionFraction:      0.25,
	TakeProfitPct:            0.08,  // 8% fixed TP — raised from 5% to give positions room to run on hourly bars
	StopLossPct:              0.04,  // 4% trailing stop — widened from 2.5% to absorb normal hourly noise without premature exits
	SessionStart:             10,    // Start trading at 10:00 to avoid early volatility and false breakouts
	SessionEnd:               16,    // Extended from 15:00 to 16:00 to allow positions a full trading day
	MinRSI:                   35,
	RequireMACDSignal:        false, // disabled: MACD signal in scoring mode caps score at 0.60 when it fails, blocking entries even with a clear pullback
	RequireBollingerBreakout: false,
	ReentryCooldownMinutes:   15,  // reduced from 30min — allows faster re-entry on hourly bars after stop-outs
	VolatilityTPMultiplier:   1.0, // 100% of Bollinger width for dynamic TP to capture larger moves during high volatility
	BreakoutLookbackBars:     1,
	RequirePriceAboveSMA:     false, // disabled: conflicts with pullback condition (price ≤ BollMiddle often means price ≤ SMA50 too)
}

var BreakoutParameters = Parameters{
	EntryMode:                "breakout",
	Timeframe:                "1h",
	MaxPositionFraction:      0.1,
	TakeProfitPct:            0.05,  // 5% — raised from 2% so positions have a meaningful target before exiting
	StopLossPct:              0.02,  // 2% — widened from 0.7% which was too tight for hourly bars on volatile stocks
	SessionStart:             9,     // Start trading at 9:00 to catch early breakouts
	SessionEnd:               16,    // Extended from 15:00 to 16:00 to allow full trading day
	MinRSI:                   55,    // Require bullish momentum for breakout entries
	RequireMACDSignal:        true,  // Require positive MACD signal for breakout entries
	RequireBollingerBreakout: true,  // Require price above upper Bollinger Band for breakout entries
	MinBollingerWidthPct:     0.007, // Minimum 0.7% Bollinger width to confirm breakout volatility
	MaxBollingerWidthPct:     0.05,  // Widened from 3% to 5% — avoids filtering out high-momentum breakouts
	ReentryCooldownMinutes:   5,     // 5-minute cooldown before re-entering after an exit to avoid overtrading on volatile breakouts
	VolatilityTPMultiplier:   1.0,
	BreakoutLookbackBars:     5, // 5-bar lookback — long enough to filter noise, short enough to catch early moves
}

var OptimizedParameters = Parameters{
	BreakoutLookbackBars:     1,
	EntryMode:                "pullback",
	Timeframe:                "1h",
	MaxBollingerWidthPct:     0.022601651405415413,
	MaxPositionFraction:      0.35322705439944385,
	MinBollingerWidthPct:     0.0047729372721378605,
	MinRSI:                   35.93228775844128,
	ReentryCooldownMinutes:   30,  // reduced from 90min — was too long on hourly bars, blocking valid re-entries
	RequireBollingerBreakout: false,
	RequireMACDSignal:        false,
	RequirePriceAboveSMA:     true,
	SessionEnd:               16,
	SessionStart:             10,  // moved earlier from 11 to recover one more hour of trading window
	StopLossPct:              0.035, // widened from 2.16% — ~1% was unreachable target given noise levels
	TakeProfitPct:            0.04,  // raised from ~1% — was too tight to hold positions through normal retracements
	VolatilityTPMultiplier:   0.7971496019830628,
}
