package tradingstrategy

import "time"

type SessionGuardDecorator struct {
	sessionStart           int
	sessionEnd             int
	reentryCooldownMinutes int
	decorated              Strategy
}

type NewSessionGuardDecoratorInput struct {
	Decorated              Strategy
	SessionStart           int
	SessionEnd             int
	ReentryCooldownMinutes int
}

func NewSessionGuardDecorator(input NewSessionGuardDecoratorInput) *SessionGuardDecorator {
	return &SessionGuardDecorator{
		decorated:              input.Decorated,
		sessionStart:           input.SessionStart,
		sessionEnd:             input.SessionEnd,
		reentryCooldownMinutes: input.ReentryCooldownMinutes,
	}
}

func (decorator *SessionGuardDecorator) Evaluate(input EvaluateInput) Decision {
	hour := input.Now.In(USMarketLocation).Hour()
	if hour < decorator.sessionStart || hour >= decorator.sessionEnd {
		return Decision{Action: ActionNone, Reason: "outside trading session window"}
	}
	if decorator.reentryCooldownMinutes > 0 && input.LastStopLossAt != nil {
		cooldownEnd := input.LastStopLossAt.Add(time.Duration(decorator.reentryCooldownMinutes) * time.Minute)
		if input.Now.Before(cooldownEnd) {
			return Decision{Action: ActionNone, Reason: "re-entry cooldown active"}
		}
	}
	return decorator.decorated.Evaluate(input)
}

