package tradingstrategy

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSessionGuardDecorator(t *testing.T) {
	Convey("Given a session guard decorator", t, func() {
		decorated := &stubStrategy{decision: Decision{Action: ActionBuy, Reason: "signal"}}
		decorator := NewSessionGuardDecorator(NewSessionGuardDecoratorInput{
			Decorated:              decorated,
			SessionStart:           10,
			SessionEnd:             15,
			ReentryCooldownMinutes: 5,
		})

		Convey("When outside the trading window", func() {
			decision := decorator.Evaluate(EvaluateInput{Now: nyTimeForTest(9, 59)})
			Convey("Then entry is blocked", func() {
				So(decorated.calls, ShouldEqual, 0)
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "outside trading session window")
			})
		})

		Convey("When cooldown is still active", func() {
			now := nyTimeForTest(11, 0)
			lastStop := now.Add(-2 * time.Minute)
			decision := decorator.Evaluate(EvaluateInput{Now: now, LastStopLossAt: &lastStop})
			Convey("Then re-entry is blocked", func() {
				So(decorated.calls, ShouldEqual, 0)
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "re-entry cooldown active")
			})
		})

		Convey("When within session and cooldown passed", func() {
			now := nyTimeForTest(11, 0)
			lastStop := now.Add(-10 * time.Minute)
			decision := decorator.Evaluate(EvaluateInput{Now: now, LastStopLossAt: &lastStop})
			Convey("Then it delegates", func() {
				So(decorated.calls, ShouldEqual, 1)
				So(decision.Action, ShouldEqual, ActionBuy)
				So(decision.Reason, ShouldEqual, "signal")
			})
		})

		Convey("When reading type", func() {
			decorated.typ = StrategyTypeScalping
			So(decorator.Type(), ShouldEqual, StrategyTypeScalping)
		})
	})
}
