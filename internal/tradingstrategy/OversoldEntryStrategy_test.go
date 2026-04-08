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
		macd := 2.0
		signal := 1.0 // macd > signal → bullish crossover

		fullInput := tradingstrategy.EvaluateInput{
			Price:            95,
			PositionQuantity: 0,
			BollLower:        &lower,
			RSI:              &rsi,
			MACD:             &macd,
			MACDSignal:       &signal,
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

		Convey("When MACD is still below signal", func() {
			input := fullInput
			lowMACD := 0.5
			input.MACD = &lowMACD
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "macd still below signal")
		})

		Convey("When bollinger data is missing", func() {
			input := fullInput
			input.BollLower = nil
			decision := strategy.Evaluate(input)
			// missing bollinger skips that check
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})

		Convey("When RSI data is missing", func() {
			input := fullInput
			input.RSI = nil
			decision := strategy.Evaluate(input)
			// missing RSI skips that check
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})
	})
}
