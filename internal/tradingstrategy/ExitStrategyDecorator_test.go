package tradingstrategy

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExitStrategyDecorator(t *testing.T) {
	Convey("Given an exit strategy decorator", t, func() {
		Convey("When flat and decorated strategy is set", func() {
			decorated := &stubStrategy{decision: Decision{Action: ActionBuy, Reason: "entry"}}
			decorator := NewExitStrategyDecorator(NewExitStrategyDecoratorInput{Decorated: decorated})
			decision := decorator.Evaluate(EvaluateInput{PositionQuantity: 0})
			Convey("Then it delegates to decorated strategy", func() {
				So(decorated.calls, ShouldEqual, 1)
				So(decision.Action, ShouldEqual, ActionBuy)
			})
		})

		Convey("When in position after session end", func() {
			decorator := NewExitStrategyDecorator(NewExitStrategyDecoratorInput{SessionEnd: 15})
			decision := decorator.Evaluate(EvaluateInput{
				PositionQuantity: 5,
				Now:              nyTimeForTest(15, 0),
			})
			Convey("Then it forces an end-of-day exit", func() {
				So(decision.Action, ShouldEqual, ActionSell)
				So(decision.Reason, ShouldEqual, "forced end-of-day exit")
				So(decision.Quantity, ShouldEqual, 5)
			})
		})

		Convey("When in position and take-profit is reached", func() {
			decorator := NewExitStrategyDecorator(NewExitStrategyDecoratorInput{
				TakeProfitPct: 0.02,
			})
			decision := decorator.Evaluate(EvaluateInput{
				PositionQuantity: 3,
				EntryPrice:       100,
				Price:            102,
				Now:              nyTimeForTest(11, 0),
			})
			Convey("Then it sells for take-profit", func() {
				So(decision.Action, ShouldEqual, ActionSell)
				So(decision.Reason, ShouldEqual, "take-profit target reached")
			})
		})

		Convey("When volatility TP is enabled and wider than fixed TP", func() {
			decorator := NewExitStrategyDecorator(NewExitStrategyDecoratorInput{
				TakeProfitPct:          0.01,
				UseVolatilityTP:        true,
				VolatilityTPMultiplier: 0.5,
			})
			decision := decorator.Evaluate(EvaluateInput{
				PositionQuantity: 2,
				EntryPrice:       100,
				Price:            102,
				BollWidthPct:     float64PtrForTest(0.10),
				Now:              nyTimeForTest(11, 0),
			})
			Convey("Then it holds because dynamic TP dominates", func() {
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "holding position")
			})
		})

		Convey("When trailing stop is triggered", func() {
			decorator := NewExitStrategyDecorator(NewExitStrategyDecoratorInput{
				StopLossPct: 0.10,
			})
			decision := decorator.Evaluate(EvaluateInput{
				PositionQuantity: 4,
				EntryPrice:       100,
				HighSinceEntry:   120,
				Price:            108,
				Now:              nyTimeForTest(11, 0),
			})
			Convey("Then it exits via trailing stop", func() {
				So(decision.Action, ShouldEqual, ActionSell)
				So(decision.Reason, ShouldEqual, "trailing stop triggered")
			})
		})

		Convey("When reading decorator type", func() {
			Convey("And a decorated strategy exists", func() {
				decorated := &stubStrategy{typ: StrategyTypePullbackTrading}
				decorator := NewExitStrategyDecorator(NewExitStrategyDecoratorInput{Decorated: decorated})
				So(decorator.Type(), ShouldEqual, StrategyTypePullbackTrading)
			})
		})
	})
}
