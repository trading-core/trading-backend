package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEntryStrategyDecorator(t *testing.T) {
	Convey("Given an entry strategy decorator", t, func() {
		Convey("When flat and a decorated strategy is configured", func() {
			decorated := &stubStrategy{
				decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"},
			}
			decorator := tradingstrategy.NewEntryStrategyDecorator(tradingstrategy.NewEntryStrategyDecoratorInput{Decorated: decorated})
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{PositionQuantity: 0})
			Convey("Then it delegates to the decorated strategy", func() {
				So(decorated.calls, ShouldEqual, 1)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldEqual, "signal")
			})
		})

		Convey("When already in position", func() {
			decorated := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy}}
			decorator := tradingstrategy.NewEntryStrategyDecorator(tradingstrategy.NewEntryStrategyDecoratorInput{Decorated: decorated})
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{PositionQuantity: 10})
			Convey("Then it blocks entry and does not delegate", func() {
				So(decorated.calls, ShouldEqual, 0)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "already in position")
			})
		})

	})
}
