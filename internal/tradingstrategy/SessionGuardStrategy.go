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
		isDailyTimeframe:       input.Timeframe == "1d" || input.Timeframe == "1w",
	}
}

func (strategy *SessionGuardStrategy) Evaluate(input EvaluateInput) Decision {
	localTimezone := input.Now.In(USMarketLocation)
	if strategy.isDailyTimeframe {
		// For daily/weekly bars the timestamp is not at market open.
		// Only gate on weekday — Saturday/Sunday bars are not trading days.
		day := localTimezone.Weekday()
		if day == time.Saturday || day == time.Sunday {
			return Decision{Action: ActionVeto, Reason: "outside trading session window"}
		}
	} else {
		// For hourly bars, check against session start/end hours.
		// Force-exit any open position at session end; otherwise veto new entries.
		hour := localTimezone.Hour()
		if hour < strategy.sessionStart || hour >= strategy.sessionEnd {
			if strategy.sessionEnd > 0 && input.PositionQuantity > 0 && hour >= strategy.sessionEnd {
				return Decision{Action: ActionSell, Reason: "forced end-of-day exit", Quantity: input.PositionQuantity}
			}
			return Decision{Action: ActionVeto, Reason: "outside trading session window"}
		}
	}
	if strategy.reentryCooldownMinutes > 0 {
		cooldown := time.Duration(strategy.reentryCooldownMinutes) * time.Minute
		if input.LastStopLossAt != nil && input.Now.Before(input.LastStopLossAt.Add(cooldown)) {
			return Decision{Action: ActionVeto, Reason: "re-entry cooldown active after stop-loss"}
		}
		if input.LastOverboughtExitAt != nil && input.Now.Before(input.LastOverboughtExitAt.Add(cooldown)) {
			return Decision{Action: ActionVeto, Reason: "re-entry cooldown active after overbought exit"}
		}
	}
	return Decision{Action: ActionNone, Reason: "within trading session window"}
}
