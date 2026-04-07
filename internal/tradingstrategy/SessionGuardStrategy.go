package tradingstrategy

import "time"

type SessionGuardStrategy struct {
	sessionStart           int
	sessionEnd             int
	reentryCooldownMinutes int
	daily                  bool // true for 1d/1w timeframes: skip hour check, use weekday check instead
}

type NewSessionGuardStrategyInput struct {
	SessionStart           int
	SessionEnd             int
	ReentryCooldownMinutes int
	Timeframe              string
}

func NewSessionGuardStrategy(input NewSessionGuardStrategyInput) *SessionGuardStrategy {
	tf := input.Timeframe
	return &SessionGuardStrategy{
		sessionStart:           input.SessionStart,
		sessionEnd:             input.SessionEnd,
		reentryCooldownMinutes: input.ReentryCooldownMinutes,
		daily:                  tf == "1d" || tf == "1w",
	}
}

func (s *SessionGuardStrategy) Evaluate(input EvaluateInput) Decision {
	local := input.Now.In(USMarketLocation)
	if s.daily {
		// For daily/weekly bars the timestamp is not at market open.
		// Only gate on weekday — Saturday/Sunday bars are not trading days.
		if wd := local.Weekday(); wd == time.Saturday || wd == time.Sunday {
			return Decision{Action: ActionVeto, Reason: "outside trading session window"}
		}
	} else {
		hour := local.Hour()
		if hour < s.sessionStart || hour >= s.sessionEnd {
			return Decision{Action: ActionVeto, Reason: "outside trading session window"}
		}
	}
	if s.reentryCooldownMinutes > 0 && input.LastStopLossAt != nil {
		cooldownEnd := input.LastStopLossAt.Add(time.Duration(s.reentryCooldownMinutes) * time.Minute)
		if input.Now.Before(cooldownEnd) {
			return Decision{Action: ActionVeto, Reason: "re-entry cooldown active"}
		}
	}
	return Decision{Action: ActionNone}
}
