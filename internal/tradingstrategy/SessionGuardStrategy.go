package tradingstrategy

import "time"

type SessionGuardStrategy struct {
	sessionStart           int
	sessionEnd             int
	reentryCooldownMinutes int
	isDailyTimeframe       bool // true for 1d/1w timeframes: skip hour check, use weekday check instead
}

type NewSessionGuardStrategyInput struct {
	SessionStart           int
	SessionEnd             int
	ReentryCooldownMinutes int
	Timeframe              string
}

func NewSessionGuardStrategy(input NewSessionGuardStrategyInput) *SessionGuardStrategy {
	return &SessionGuardStrategy{
		sessionStart:           input.SessionStart,
		sessionEnd:             input.SessionEnd,
		reentryCooldownMinutes: input.ReentryCooldownMinutes,
		isDailyTimeframe:       input.Timeframe == "1d",
	}
}

func (strategy *SessionGuardStrategy) Evaluate(input EvaluateInput) Decision {
	localTimezone := input.Now.In(USMarketLocation)
	// Force-exit any open position at session end regardless of timeframe.
	if strategy.sessionEnd > 0 && input.PositionQuantity > 0 {
		if localTimezone.Hour() >= strategy.sessionEnd {
			return Decision{Action: ActionSell, Reason: "forced end-of-day exit", Quantity: input.PositionQuantity}
		}
	}
	if strategy.isDailyTimeframe {
		// For daily/weekly bars the timestamp is not at market open.
		// Only gate on weekday — Saturday/Sunday bars are not trading days.
		day := localTimezone.Weekday()
		isWeekend := day == time.Saturday || day == time.Sunday
		if isWeekend {
			return Decision{Action: ActionVeto, Reason: "outside trading session window"}
		}
	} else {
		// For hourly bars, we check against session start/end hours.
		hour := localTimezone.Hour()
		if hour < strategy.sessionStart || hour >= strategy.sessionEnd {
			return Decision{Action: ActionVeto, Reason: "outside trading session window"}
		}
	}
	if strategy.reentryCooldownMinutes > 0 && input.LastStopLossAt != nil {
		cooldownEnd := input.LastStopLossAt.Add(time.Duration(strategy.reentryCooldownMinutes) * time.Minute)
		if input.Now.Before(cooldownEnd) {
			return Decision{Action: ActionVeto, Reason: "re-entry cooldown active"}
		}
	}
	return Decision{Action: ActionNone, Reason: "within trading session window"}
}
