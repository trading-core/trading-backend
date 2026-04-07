package tradingstrategy_test

import (
	"testing"
	"time"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSessionGuardStrategy(t *testing.T) {
	Convey("Given an intraday session guard strategy", t, func() {
		strategy := tradingstrategy.NewSessionGuardStrategy(tradingstrategy.NewSessionGuardStrategyInput{
			SessionStart:           10,
			SessionEnd:             15,
			ReentryCooldownMinutes: 5,
			Timeframe:              "1h",
		})

		Convey("When outside the trading window", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Now: nyTimeForTest(9, 59)})
			Convey("Then it vetoes", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "outside trading session window")
			})
		})

		Convey("When cooldown is still active", func() {
			now := nyTimeForTest(11, 0)
			lastStop := now.Add(-2 * time.Minute)
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Now: now, LastStopLossAt: &lastStop})
			Convey("Then it vetoes", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "re-entry cooldown active")
			})
		})

		Convey("When within session and cooldown passed", func() {
			now := nyTimeForTest(11, 0)
			lastStop := now.Add(-10 * time.Minute)
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Now: now, LastStopLossAt: &lastStop})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})
	})

	Convey("Given a daily session guard strategy", t, func() {
		strategy := tradingstrategy.NewSessionGuardStrategy(tradingstrategy.NewSessionGuardStrategyInput{
			SessionStart: 9,
			SessionEnd:   16,
			Timeframe:    "1d",
		})

		Convey("When the bar timestamp is 8 PM ET on a Monday (outside intraday window)", func() {
			// Alpaca daily bars are often stamped at the close (4 PM ET) or
			// end-of-day, which is well outside the intraday 9-16 window.
			// Hour check must be skipped — only weekday check applies.
			monday8pm := nyTimeForTest(20, 0) // 2026-04-06 is a Monday
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Now: monday8pm})
			Convey("Then it abstains (weekday passes)", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("When the bar falls on a Saturday in ET", func() {
			saturday := time.Date(2026, time.April, 4, 12, 0, 0, 0, tradingstrategy.USMarketLocation)
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Now: saturday})
			Convey("Then it vetoes", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "outside trading session window")
			})
		})
	})
}
