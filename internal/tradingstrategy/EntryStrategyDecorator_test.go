package tradingstrategy

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEntryStrategyDecorator(t *testing.T) {
	Convey("Given an entry strategy decorator", t, func() {
		Convey("When flat and a decorated strategy is configured", func() {
			decorated := &stubStrategy{
				typ:      StrategyTypePullbackTrading,
				decision: Decision{Action: ActionBuy, Reason: "signal"},
			}
			decorator := NewEntryStrategyDecorator(NewEntryStrategyDecoratorInput{Decorated: decorated})
			decision := decorator.Evaluate(EvaluateInput{PositionQuantity: 0})
			Convey("Then it delegates to the decorated strategy", func() {
				So(decorated.calls, ShouldEqual, 1)
				So(decision.Action, ShouldEqual, ActionBuy)
				So(decision.Reason, ShouldEqual, "signal")
			})
		})

		Convey("When already in position", func() {
			decorated := &stubStrategy{decision: Decision{Action: ActionBuy}}
			decorator := NewEntryStrategyDecorator(NewEntryStrategyDecoratorInput{Decorated: decorated})
			decision := decorator.Evaluate(EvaluateInput{PositionQuantity: 10})
			Convey("Then it blocks entry and does not delegate", func() {
				So(decorated.calls, ShouldEqual, 0)
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "already in position")
			})
		})

		Convey("When reading decorator type", func() {
			Convey("And a decorated strategy exists", func() {
				decorated := &stubStrategy{typ: StrategyTypeBreakoutTrading}
				decorator := NewEntryStrategyDecorator(NewEntryStrategyDecoratorInput{Decorated: decorated})
				So(decorator.Type(), ShouldEqual, StrategyTypeBreakoutTrading)
			})
		})
	})
}
