package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMeanReversionExitStrategy(t *testing.T) {
	bollMiddle := 105.0

	Convey("Given a mean reversion exit strategy (enabled)", t, func() {
		strategy := tradingstrategy.NewMeanReversionExitStrategy(tradingstrategy.NewMeanReversionExitStrategyInput{
			Enabled: true,
		})
		fullInput := tradingstrategy.EvaluateInput{
			Price:            105,
			PositionQuantity: 10,
			BollMiddle:       &bollMiddle,
		}

		Convey("When price equals the Bollinger middle, exit fires", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			So(decision.Reason, ShouldEqual, "mean reversion exit: price reached bollinger middle")
			So(decision.Quantity, ShouldEqual, 10)
		})

		Convey("When price is above the Bollinger middle, exit fires", func() {
			input := fullInput
			input.Price = 110
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
		})

		Convey("When price is below the Bollinger middle, exit does not fire", func() {
			input := fullInput
			input.Price = 100
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "mean reversion exit: price below bollinger middle")
		})

		Convey("When BollMiddle is unavailable, exit does not fire", func() {
			input := fullInput
			input.BollMiddle = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "mean reversion exit: boll_middle unavailable")
		})

		Convey("When entry price is below the Bollinger middle (range-mode entry), exit fires", func() {
			input := fullInput
			input.EntryPrice = 98 // entered below mean
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
		})

		Convey("When entry price is at the Bollinger middle (trend-mode entry), exit does not fire", func() {
			input := fullInput
			input.EntryPrice = 105 // entered at the mean — treat as trend entry
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "mean reversion exit: entry price at or above bollinger middle")
		})

		Convey("When entry price is above the Bollinger middle (trend-mode entry), exit does not fire", func() {
			input := fullInput
			input.EntryPrice = 108 // entered above mean — trend entry
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "mean reversion exit: entry price at or above bollinger middle")
		})

		Convey("When not holding a position, exit does not fire", func() {
			input := fullInput
			input.PositionQuantity = 0
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})

	Convey("Given a mean reversion exit strategy (disabled)", t, func() {
		strategy := tradingstrategy.NewMeanReversionExitStrategy(tradingstrategy.NewMeanReversionExitStrategyInput{
			Enabled: false,
		})
		Convey("Then it always abstains regardless of price", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:            110,
				PositionQuantity: 10,
				BollMiddle:       &bollMiddle,
			})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})
}
