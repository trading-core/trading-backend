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

		Convey("Given stop confirmation: price must be below Bollinger middle", func() {
			middle := 110.0
			strategy := tradingstrategy.NewTrailingStopStrategy(tradingstrategy.NewTrailingStopStrategyInput{
				StopLossPct:               0.10,
				StopConfirmBelowBollMiddle: true,
			})
			baseInput := tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            107, // below stop level (120 * 0.90 = 108)
				BollMiddle:       &middle,
			}

			Convey("When price is below Bollinger middle, stop fires", func() {
				input := baseInput
				belowMiddle := 109.0
				input.BollMiddle = &belowMiddle
				// price 107 < bollMiddle 109 → confirmed
				decision := strategy.Evaluate(input)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			})

			Convey("When price is above Bollinger middle, stop is suppressed", func() {
				decision := strategy.Evaluate(baseInput) // price 107 < middle 110? no, 107 < 110 → confirmed... wait
				// price=107, middle=110: 107 < 110 so it IS below middle → fires
				// Let me use a middle below price to suppress
				abovePrice := 106.0
				input := baseInput
				input.BollMiddle = &abovePrice // price 107 >= middle 106 → not below middle → suppressed
				decision = strategy.Evaluate(input)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldContainSubstring, "bollinger middle")
			})

			Convey("When Bollinger middle data is missing, stop is suppressed", func() {
				input := baseInput
				input.BollMiddle = nil
				decision := strategy.Evaluate(input)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("Given stop confirmation: RSI must be below threshold", func() {
			weakRSI := 42.0
			strongRSI := 55.0
			strategy := tradingstrategy.NewTrailingStopStrategy(tradingstrategy.NewTrailingStopStrategyInput{
				StopLossPct:         0.10,
				StopConfirmRSIBelow: 50.0,
			})
			baseInput := tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            107, // below stop level
			}

			Convey("When RSI is below threshold, stop fires", func() {
				input := baseInput
				input.RSI = &weakRSI
				decision := strategy.Evaluate(input)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			})

			Convey("When RSI is above threshold, stop is suppressed", func() {
				input := baseInput
				input.RSI = &strongRSI
				decision := strategy.Evaluate(input)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldContainSubstring, "rsi not confirming weakness")
			})

			Convey("When RSI data is missing, stop is suppressed", func() {
				decision := strategy.Evaluate(baseInput)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("Given both confirmations are required", func() {
			middle := 110.0
			weakRSI := 42.0
			strategy := tradingstrategy.NewTrailingStopStrategy(tradingstrategy.NewTrailingStopStrategyInput{
				StopLossPct:               0.10,
				StopConfirmBelowBollMiddle: true,
				StopConfirmRSIBelow:        50.0,
			})
			baseInput := tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            107,
				BollMiddle:       &middle,
				RSI:              &weakRSI,
			}

			Convey("When both confirmations pass, stop fires", func() {
				decision := strategy.Evaluate(baseInput)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			})

			Convey("When only RSI confirms but Bollinger middle is missing, stop is suppressed", func() {
				input := baseInput
				input.BollMiddle = nil
				decision := strategy.Evaluate(input)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})

			Convey("When only Bollinger confirms but RSI is too strong, stop is suppressed", func() {
				strongRSI := 60.0
				input := baseInput
				input.RSI = &strongRSI
				decision := strategy.Evaluate(input)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})
	})
}
