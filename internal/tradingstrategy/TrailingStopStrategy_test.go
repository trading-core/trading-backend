package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTrailingStopStrategy(t *testing.T) {
	Convey("Given a trailing stop strategy", t, func() {
		Convey("When flat", func() {
			strategy := tradingstrategy.NewTrailingStopStrategy(tradingstrategy.NewTrailingStopStrategyInput{StopLossPct: 0.10})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{PositionQuantity: 0})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("When in position and trailing stop is triggered", func() {
			strategy := tradingstrategy.NewTrailingStopStrategy(tradingstrategy.NewTrailingStopStrategyInput{StopLossPct: 0.10})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            108, // below 120 * 0.90 = 108 (at the boundary)
			})
			Convey("Then it exits via trailing stop", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldEqual, "trailing stop triggered")
				So(decision.Quantity, ShouldEqual, 4)
			})
		})

		Convey("When in position and price is above stop level", func() {
			strategy := tradingstrategy.NewTrailingStopStrategy(tradingstrategy.NewTrailingStopStrategyInput{StopLossPct: 0.10})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            115,
			})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})
	})
}
