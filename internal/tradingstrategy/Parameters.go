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
	MaxPositionFraction:      0.25,
	TakeProfitPct:            0.05,  // 5% fixed TP
	StopLossPct:              0.025, // 2.5% trailing stop — gives mean-reversion room to dip before bouncing
	SessionStart:             10,    // Start trading at 10:00 to avoid early volatility and false breakouts
	SessionEnd:               15,    // Exit all positions by 15:00 to avoid end-of-day risk
	MinRSI:                   35,
	RequireMACDSignal:        false, // disabled: MACD signal in scoring mode caps score at 0.60 when it fails, blocking entries even with a clear pullback
	RequireBollingerBreakout: false,
	ReentryCooldownMinutes:   30,  // reduced: 90min was too long on hourly bars, missing recovery setups after stop-outs
	VolatilityTPMultiplier:   1.0, // 100% of Bollinger width for dynamic TP to capture larger moves during high volatility
	BreakoutLookbackBars:     1,
	RequirePriceAboveSMA:     false, // disabled: conflicts with pullback condition (price ≤ BollMiddle often means price ≤ SMA50 too)
}

var BreakoutParameters = Parameters{
	EntryMode:                "breakout",
	MaxPositionFraction:      0.1,
	TakeProfitPct:            0.02,  // 2%
	StopLossPct:              0.007, // 0.7%
	SessionStart:             9,     // Start trading at 9:00 to catch early breakouts
	SessionEnd:               15,    // Exit all positions by 15:00 to avoid end-of-day risk
	MinRSI:                   55,    // Require bullish momentum for breakout entries
	RequireMACDSignal:        true,  // Require positive MACD signal for breakout entries
	RequireBollingerBreakout: true,  // Require price above upper Bollinger Band for breakout entries
	MinBollingerWidthPct:     0.007, // Minimum 0.7% Bollinger width to confirm breakout volatility
	MaxBollingerWidthPct:     0.03,  // Maximum 3% Bollinger width to avoid extreme volatility
	ReentryCooldownMinutes:   5,     // 5-minute cooldown before re-entering after an exit to avoid overtrading on volatile breakouts
	VolatilityTPMultiplier:   1.0,
	BreakoutLookbackBars:     5, // 5-bar lookback — long enough to filter noise, short enough to catch early moves
}

var OptimizedParameters = Parameters{
	BreakoutLookbackBars:     1,
	EntryMode:                "pullback",
	MaxBollingerWidthPct:     0.022601651405415413,
	MaxPositionFraction:      0.35322705439944385,
	MinBollingerWidthPct:     0.0047729372721378605,
	MinRSI:                   35.93228775844128,
	ReentryCooldownMinutes:   90,
	RequireBollingerBreakout: false,
	RequireMACDSignal:        false,
	RequirePriceAboveSMA:     true,
	SessionEnd:               16,
	SessionStart:             11,
	StopLossPct:              0.021593698844776812,
	TakeProfitPct:            0.009812974895691324,
	VolatilityTPMultiplier:   0.7971496019830628,
}
