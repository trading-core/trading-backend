package tradingstrategy_test

import (
	"testing"
	"time"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSessionGuardDecorator(t *testing.T) {
	Convey("Given a session guard decorator", t, func() {
		decorated := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"}}
		decorator := tradingstrategy.NewSessionGuardDecorator(tradingstrategy.NewSessionGuardDecoratorInput{
			Decorated:              decorated,
			SessionStart:           10,
			SessionEnd:             15,
			ReentryCooldownMinutes: 5,
		})

		Convey("When outside the trading window", func() {
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Now: nyTimeForTest(9, 59)})
			Convey("Then entry is blocked", func() {
				So(decorated.calls, ShouldEqual, 0)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "outside trading session window")
			})
		})

		Convey("When cooldown is still active", func() {
			now := nyTimeForTest(11, 0)
			lastStop := now.Add(-2 * time.Minute)
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Now: now, LastStopLossAt: &lastStop})
			Convey("Then re-entry is blocked", func() {
				So(decorated.calls, ShouldEqual, 0)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "re-entry cooldown active")
			})
		})

		Convey("When within session and cooldown passed", func() {
			now := nyTimeForTest(11, 0)
			lastStop := now.Add(-10 * time.Minute)
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Now: now, LastStopLossAt: &lastStop})
			Convey("Then it delegates", func() {
				So(decorated.calls, ShouldEqual, 1)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldEqual, "signal")
			})
		})

	})
}

type stubStrategy struct {
	decision tradingstrategy.Decision
	calls    int
}

func (strategy *stubStrategy) Evaluate(input tradingstrategy.EvaluateInput) tradingstrategy.Decision {
	strategy.calls++
	return strategy.decision
}

func float64PtrForTest(value float64) *float64 {
	return &value
}

func nyTimeForTest(hour int, minute int) time.Time {
	return time.Date(2026, time.April, 6, hour, minute, 0, 0, tradingstrategy.USMarketLocation)
}
