package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMACDReversalExitStrategy(t *testing.T) {
	macdBelowSignal := -0.5
	macdAboveSignal := 0.5
	signal := 0.0

	Convey("Given a MACD reversal exit strategy (enabled)", t, func() {
		strategy := tradingstrategy.NewMACDReversalExitStrategy(tradingstrategy.NewMACDReversalExitStrategyInput{
			Enabled: true,
		})
		fullInput := tradingstrategy.EvaluateInput{
			Price:            110,
			EntryPrice:       100,
			PositionQuantity: 5,
			MACD:             &macdBelowSignal,
			MACDSignal:       &signal,
		}

		Convey("When MACD is below signal and price is above entry, exit fires", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			So(decision.Reason, ShouldEqual, "macd reversal exit: macd crossed below signal above entry price")
			So(decision.Quantity, ShouldEqual, 5)
		})

		Convey("When MACD is above signal, exit does not fire", func() {
			input := fullInput
			input.MACD = &macdAboveSignal
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "macd reversal exit: macd above signal")
		})

		Convey("When MACD equals signal, exit does not fire", func() {
			input := fullInput
			input.MACD = &signal
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "macd reversal exit: macd above signal")
		})

		Convey("When price is at or below entry price, exit does not fire (guards range-mode trades)", func() {
			input := fullInput
			input.Price = fullInput.EntryPrice
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "macd reversal exit: price at or below entry price")
		})

		Convey("When MACD data is unavailable, exit does not fire", func() {
			input := fullInput
			input.MACD = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "macd reversal exit: macd unavailable")
		})

		Convey("When MACDSignal data is unavailable, exit does not fire", func() {
			input := fullInput
			input.MACDSignal = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "macd reversal exit: macd unavailable")
		})

		Convey("When not holding a position, exit does not fire", func() {
			input := fullInput
			input.PositionQuantity = 0
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})

	Convey("Given a MACD reversal exit strategy (disabled)", t, func() {
		strategy := tradingstrategy.NewMACDReversalExitStrategy(tradingstrategy.NewMACDReversalExitStrategyInput{
			Enabled: false,
		})
		Convey("Then it always abstains", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:            110,
				EntryPrice:       100,
				PositionQuantity: 5,
				MACD:             &macdBelowSignal,
				MACDSignal:       &signal,
			})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})
}
