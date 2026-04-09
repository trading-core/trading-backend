package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestATRStopStrategy(t *testing.T) {
	atr := 6.0 // ATRMultiplier=2.0 → stopLevel = highSinceEntry − 12

	Convey("Given an ATR stop strategy", t, func() {
		Convey("When flat", func() {
			strategy := tradingstrategy.NewATRStopStrategy(tradingstrategy.NewATRStopStrategyInput{ATRMultiplier: 2.0})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{PositionQuantity: 0, ATR: &atr})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("When in position and stop is triggered", func() {
			strategy := tradingstrategy.NewATRStopStrategy(tradingstrategy.NewATRStopStrategyInput{ATRMultiplier: 2.0})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            107, // stopLevel = 120 − (2×6) = 108; 107 ≤ 108 → fires
				ATR:              &atr,
			})
			Convey("Then it exits and the reason names the prices", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldContainSubstring, "atr stop:")
				So(decision.Reason, ShouldContainSubstring, "107.00")
				So(decision.Reason, ShouldContainSubstring, "108.00")
				So(decision.Quantity, ShouldEqual, 4)
			})
		})

		Convey("When price is exactly at the stop level it triggers", func() {
			strategy := tradingstrategy.NewATRStopStrategy(tradingstrategy.NewATRStopStrategyInput{ATRMultiplier: 2.0})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            108, // exactly at stopLevel → fires
				ATR:              &atr,
			})
			Convey("Then it exits", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			})
		})

		Convey("When price is above stop level", func() {
			strategy := tradingstrategy.NewATRStopStrategy(tradingstrategy.NewATRStopStrategyInput{ATRMultiplier: 2.0})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            115, // 115 > 108 → no trigger
				ATR:              &atr,
			})
			Convey("Then it abstains and the reason names the prices", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldContainSubstring, "atr stop:")
				So(decision.Reason, ShouldContainSubstring, "115.00")
				So(decision.Reason, ShouldContainSubstring, "108.00")
			})
		})

		Convey("When ATR data is missing, stop is suppressed", func() {
			strategy := tradingstrategy.NewATRStopStrategy(tradingstrategy.NewATRStopStrategyInput{ATRMultiplier: 2.0})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            107,
				ATR:              nil,
			})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "atr stop: atr unavailable")
			})
		})

		Convey("When ATRMultiplier is zero, stop is disabled", func() {
			strategy := tradingstrategy.NewATRStopStrategy(tradingstrategy.NewATRStopStrategyInput{ATRMultiplier: 0})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            50,
				ATR:              &atr,
			})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})
	})
}
