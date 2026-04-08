package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestOverboughtExitStrategy(t *testing.T) {
	Convey("Given an overbought exit strategy", t, func() {
		strategy := tradingstrategy.NewOverboughtExitStrategy(tradingstrategy.NewOverboughtExitStrategyInput{
			OverboughtRSI: 70,
		})

		upper := 100.0
		rsi := 75.0
		fullInput := tradingstrategy.EvaluateInput{
			Price:            105,
			PositionQuantity: 10,
			BollUpper:        &upper,
			RSI:              &rsi,
		}

		Convey("When not holding a position", func() {
			input := fullInput
			input.PositionQuantity = 0
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})

		Convey("When all overbought conditions are met", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			So(decision.Quantity, ShouldEqual, 10)
		})

		Convey("When price is below upper bollinger", func() {
			input := fullInput
			below := 110.0 // upper band above price
			input.BollUpper = &below
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "price below upper bollinger")
		})

		Convey("When RSI is not overbought", func() {
			input := fullInput
			lowRSI := 60.0
			input.RSI = &lowRSI
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi not overbought")
		})

		Convey("When MACD is above signal, exit fires anyway (MACD lags at peaks)", func() {
			input := fullInput
			macd := 3.0
			signal := 1.0
			input.MACD = &macd
			input.MACDSignal = &signal
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
		})

		Convey("When bollinger data is missing", func() {
			input := fullInput
			input.BollUpper = nil
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
}
