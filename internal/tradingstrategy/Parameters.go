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
	RequireBollingerSqueeze  bool    `json:"require_bollinger_squeeze,omitempty"`  // if true, require Bollinger Band width below MinBollingerWidthPct to enter
	MaxBollingerWidthPct     float64 `json:"max_bollinger_width_pct,omitempty"`    // maximum Bollinger Band width as percentage to enter (e.g. 0.02 for 2%)
	ReentryCooldownMinutes   int     `json:"reentry_cooldown_minutes,omitempty"`   // cooldown period in minutes before re-entering after an exit
	UseVolatilityTP          bool    `json:"use_volatility_tp,omitempty"`          // if true, take-profit is max of fixed TP and volatility-based TP
	VolatilityTPMultiplier   float64 `json:"volatility_tp_multiplier,omitempty"`   // multiplier for Bollinger width to calculate dynamic TP (e.g. 0.5 for 50% of Bollinger width)
	RiskPerTradePct          float64 `json:"risk_per_trade_pct,omitempty"`         // percentage of account equity to risk per trade (e.g. 0.01 for 1%). Used to calculate position size based on stop-loss distance.
	BreakoutLookbackBars     int     `json:"breakout_lookback_bars,omitempty"`     // number of bars to lookback for breakout (1=session high, 5=5-bar high). Default 1.
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
	tradingStrategy = NewEntryStrategyDecorator(NewEntryStrategyDecoratorInput{
		Decorated: tradingStrategy,
	})
	tradingStrategy = NewIndicatorFilterDecorator(NewIndicatorFilterDecoratorInput{
		Decorated:                tradingStrategy,
		MinRSI:                   parameters.MinRSI,
		RequireMACDSignal:        parameters.RequireMACDSignal,
		RequireBollingerBreakout: parameters.RequireBollingerBreakout,
		MinBollingerWidthPct:     parameters.MinBollingerWidthPct,
		RequireBollingerSqueeze:  parameters.RequireBollingerSqueeze,
		MaxBollingerWidthPct:     parameters.MaxBollingerWidthPct,
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
		RiskPerTradePct:     parameters.RiskPerTradePct,
		StopLossPct:         parameters.StopLossPct,
	})
	tradingStrategy = NewExitStrategyDecorator(NewExitStrategyDecoratorInput{
		Decorated:              tradingStrategy,
		SessionEnd:             parameters.SessionEnd,
		TakeProfitPct:          parameters.TakeProfitPct,
		StopLossPct:            parameters.StopLossPct,
		UseVolatilityTP:        parameters.UseVolatilityTP,
		VolatilityTPMultiplier: parameters.VolatilityTPMultiplier,
	})
	tradingStrategy = NewSystemGuardDecorator(NewSystemGuardDecoratorInput{
		Decorated: tradingStrategy,
	})
	return tradingStrategy
}

var PullbackParameters = Parameters{
	EntryMode:                "pullback",
	MaxPositionFraction:      0.1,
	TakeProfitPct:            0.01,  // 1% fixed TP (overridden by volatility-based TP if enabled)
	StopLossPct:              0.005, // 0.5% trailing stop
	SessionStart:             10,    // Start trading at 10:00 to avoid early volatility and false breakouts
	SessionEnd:               15,    // Exit all positions by 15:00 to avoid end-of-day risk
	MinRSI:                   35,
	RequireMACDSignal:        true,
	RequireBollingerBreakout: false,
	MinBollingerWidthPct:     0,
	RequireBollingerSqueeze:  false,
	MaxBollingerWidthPct:     0.015, // 1.5% Bollinger width filter to avoid low-volatility breakouts
	ReentryCooldownMinutes:   10,    // 10-minute cooldown before re-entering after an exit
	UseVolatilityTP:          false,
	VolatilityTPMultiplier:   0.5,
	RiskPerTradePct:          0,
	BreakoutLookbackBars:     1,
}

var BreakoutParameters = Parameters{
	EntryMode:                "breakout",
	MaxPositionFraction:      0.1,
	RiskPerTradePct:          0,     // upgrade later
	TakeProfitPct:            0.02,  // 2%
	StopLossPct:              0.007, // 0.7%
	SessionStart:             9,     // Start trading at 9:00 to catch early breakouts
	SessionEnd:               15,    // Exit all positions by 15:00 to avoid end-of-day risk
	MinRSI:                   55,    // Require bullish momentum for breakout entries
	RequireMACDSignal:        true,  // Require positive MACD signal for breakout entries
	RequireBollingerBreakout: true,  // Require price above upper Bollinger Band for breakout entries
	RequireBollingerSqueeze:  true,  // Require Bollinger Band squeeze (low volatility) before breakout to filter out false breakouts
	MinBollingerWidthPct:     0.007, // Minimum 0.7% Bollinger width to confirm breakout volatility
	MaxBollingerWidthPct:     0.03,  // Maximum 3% Bollinger width to avoid extreme volatility
	ReentryCooldownMinutes:   5,     // 5-minute cooldown before re-entering after an exit to avoid overtrading on volatile breakouts
	UseVolatilityTP:          true,  // Use volatility-based TP to capture larger moves during high-volatility breakouts
	VolatilityTPMultiplier:   1.0,
	BreakoutLookbackBars:     15,
}
