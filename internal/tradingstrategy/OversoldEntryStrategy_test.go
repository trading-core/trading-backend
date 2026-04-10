package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestOversoldEntryStrategy(t *testing.T) {
	Convey("Given an oversold entry strategy", t, func() {
		strategy := tradingstrategy.NewOversoldEntryStrategy(tradingstrategy.NewOversoldEntryStrategyInput{
			OversoldRSI: 30,
		})

		lower := 100.0
		rsi := 25.0

		fullInput := tradingstrategy.EvaluateInput{
			Price:            95,
			PositionQuantity: 0,
			BollLower:        &lower,
			RSI:              &rsi,
		}

		Convey("When already in a position", func() {
			input := fullInput
			input.PositionQuantity = 5
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})

		Convey("When all oversold conditions are met", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})

		Convey("When price is above lower bollinger", func() {
			input := fullInput
			above := 90.0 // lower band below price
			input.BollLower = &above
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "price above lower bollinger")
		})

		Convey("When RSI is not oversold", func() {
			input := fullInput
			highRSI := 45.0
			input.RSI = &highRSI
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi not oversold")
		})

		Convey("When MACD is below signal it still fires (MACD not required for mean-reversion)", func() {
			input := fullInput
			lowMACD := 0.5
			highSignal := 1.0
			input.MACD = &lowMACD
			input.MACDSignal = &highSignal
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})

		Convey("When bollinger data is missing", func() {
			input := fullInput
			input.BollLower = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "bollinger unavailable")
		})

		Convey("When RSI data is missing", func() {
			input := fullInput
			input.RSI = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi unavailable")
		})
	})

	Convey("Given an oversold entry strategy with RSI check disabled", t, func() {
		strategy := tradingstrategy.NewOversoldEntryStrategy(tradingstrategy.NewOversoldEntryStrategyInput{})

		lower := 100.0
		fullInput := tradingstrategy.EvaluateInput{
			Price:            95,
			PositionQuantity: 0,
			BollLower:        &lower,
		}

		Convey("When price is at or below the lower bollinger, entry fires on bollinger alone", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
			So(decision.Reason, ShouldEqual, "oversold entry: lower bollinger")
		})

		Convey("When RSI data is missing, entry still fires", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})
	})
}
