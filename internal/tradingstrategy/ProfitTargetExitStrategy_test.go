package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestProfitTargetExitStrategy(t *testing.T) {
	atr := 5.0 // multiplier=3.0 → target = entry + 15

	Convey("Given a profit target exit strategy with multiplier 3.0", t, func() {
		strategy := tradingstrategy.NewProfitTargetExitStrategy(tradingstrategy.NewProfitTargetExitStrategyInput{
			ProfitTargetMultiplier: 3.0,
		})
		fullInput := tradingstrategy.EvaluateInput{
			Price:            115,
			EntryPrice:       100,
			PositionQuantity: 8,
			ATR:              &atr,
		}

		Convey("When price is at the target (entry + 3×ATR), exit fires", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			So(decision.Reason, ShouldContainSubstring, "profit target:")
			So(decision.Reason, ShouldContainSubstring, "115.00")
			So(decision.Quantity, ShouldEqual, 8)
		})

		Convey("When price is above the target, exit fires", func() {
			input := fullInput
			input.Price = 120
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
		})

		Convey("When price is below the target, exit does not fire", func() {
			input := fullInput
			input.Price = 114
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldContainSubstring, "profit target:")
			So(decision.Reason, ShouldContainSubstring, "114.00")
		})

		Convey("When ATR is unavailable, exit does not fire", func() {
			input := fullInput
			input.ATR = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "profit target: atr unavailable")
		})

		Convey("When not holding a position, exit does not fire", func() {
			input := fullInput
			input.PositionQuantity = 0
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})

		Convey("When EntryPrice is zero (no recorded entry), exit does not fire", func() {
			input := fullInput
			input.EntryPrice = 0
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})

	Convey("Given a profit target exit strategy with multiplier 0 (disabled)", t, func() {
		strategy := tradingstrategy.NewProfitTargetExitStrategy(tradingstrategy.NewProfitTargetExitStrategyInput{
			ProfitTargetMultiplier: 0,
		})
		Convey("Then it always abstains", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:            200,
				EntryPrice:       100,
				PositionQuantity: 5,
				ATR:              &atr,
			})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})
}
