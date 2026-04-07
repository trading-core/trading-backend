package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExitStrategyDecorator(t *testing.T) {
	Convey("Given an exit strategy decorator", t, func() {
		Convey("When flat and decorated strategy is set", func() {
			decorated := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "entry"}}
			decorator := tradingstrategy.NewExitStrategyDecorator(tradingstrategy.NewExitStrategyDecoratorInput{Decorated: decorated})
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{PositionQuantity: 0})
			Convey("Then it delegates to decorated strategy", func() {
				So(decorated.calls, ShouldEqual, 1)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
			})
		})

		Convey("When in position after session end", func() {
			decorator := tradingstrategy.NewExitStrategyDecorator(tradingstrategy.NewExitStrategyDecoratorInput{SessionEnd: 15})
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 5,
				Now:              nyTimeForTest(15, 0),
			})
			Convey("Then it forces an end-of-day exit", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldEqual, "forced end-of-day exit")
				So(decision.Quantity, ShouldEqual, 5)
			})
		})

		Convey("When in position and take-profit is reached", func() {
			decorator := tradingstrategy.NewExitStrategyDecorator(tradingstrategy.NewExitStrategyDecoratorInput{
				TakeProfitPct: 0.02,
			})
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 3,
				EntryPrice:       100,
				Price:            102,
				Now:              nyTimeForTest(11, 0),
			})
			Convey("Then it sells for take-profit", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldEqual, "take-profit target reached")
			})
		})

		Convey("When volatility TP is enabled and wider than fixed TP", func() {
			decorator := tradingstrategy.NewExitStrategyDecorator(tradingstrategy.NewExitStrategyDecoratorInput{
				TakeProfitPct:          0.01,
				VolatilityTPMultiplier: 0.5,
			})
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 2,
				EntryPrice:       100,
				Price:            102,
				BollWidthPct:     float64PtrForTest(0.10),
				Now:              nyTimeForTest(11, 0),
			})
			Convey("Then it holds because dynamic TP dominates", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "holding position")
			})
		})

		Convey("When RSI hits overbought level", func() {
			rsi := 72.0
			decorator := tradingstrategy.NewExitStrategyDecorator(tradingstrategy.NewExitStrategyDecoratorInput{
				OverboughtRSI: 70,
			})
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 3,
				EntryPrice:       100,
				Price:            105,
				RSI:              &rsi,
				Now:              nyTimeForTest(11, 0),
			})
			Convey("Then it exits via RSI overbought signal", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldEqual, "rsi overbought")
				So(decision.Quantity, ShouldEqual, 3)
			})
		})

		Convey("When trailing stop is triggered", func() {
			decorator := tradingstrategy.NewExitStrategyDecorator(tradingstrategy.NewExitStrategyDecoratorInput{
				StopLossPct: 0.10,
			})
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            108,
				Now:              nyTimeForTest(11, 0),
			})
			Convey("Then it exits via trailing stop", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldEqual, "trailing stop triggered")
			})
		})

	})
}
