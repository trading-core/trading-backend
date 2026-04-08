package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTakeProfitStrategy(t *testing.T) {
	Convey("Given a take-profit strategy", t, func() {
		Convey("When flat", func() {
			strategy := tradingstrategy.NewTakeProfitStrategy(tradingstrategy.NewTakeProfitStrategyInput{TakeProfitPct: 0.02})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{PositionQuantity: 0})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("When in position and take-profit is reached", func() {
			strategy := tradingstrategy.NewTakeProfitStrategy(tradingstrategy.NewTakeProfitStrategyInput{TakeProfitPct: 0.02})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 3,
				EntryPrice:       100,
				Price:            102,
			})
			Convey("Then it sells for take-profit", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldEqual, "take-profit target reached")
				So(decision.Quantity, ShouldEqual, 3)
			})
		})

		Convey("When volatility TP dominates and price has not reached dynamic target", func() {
			strategy := tradingstrategy.NewTakeProfitStrategy(tradingstrategy.NewTakeProfitStrategyInput{
				TakeProfitPct:          0.01,
				VolatilityTPMultiplier: 0.5,
			})
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 2,
				EntryPrice:       100,
				Price:            102,
				BollWidthPct:     float64PtrForTest(0.10), // dynamic TP = 5%, price only up 2%
			})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})
	})
}
