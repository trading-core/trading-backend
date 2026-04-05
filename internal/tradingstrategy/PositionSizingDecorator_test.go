package tradingstrategy

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPositionSizingDecorator(t *testing.T) {
	Convey("Given a position sizing decorator", t, func() {
		Convey("When there is no buying power", func() {
			decorator := NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
				Decorated:           &stubStrategy{decision: Decision{Action: ActionBuy, Reason: "signal"}},
				MaxPositionFraction: 0.1,
				StopLossPct:         0.02,
			})
			decision := decorator.Evaluate(EvaluateInput{Price: 100})
			Convey("Then it blocks entry", func() {
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "no buying power available")
			})
		})

		Convey("When decorated strategy returns non-buy", func() {
			decorated := &stubStrategy{decision: Decision{Action: ActionNone, Reason: "no entry signal"}}
			decorator := NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
				Decorated:           decorated,
				MaxPositionFraction: 0.1,
				StopLossPct:         0.02,
			})
			decision := decorator.Evaluate(EvaluateInput{Price: 100, BuyingPower: 1000})
			Convey("Then it passes through unchanged", func() {
				So(decorated.calls, ShouldEqual, 1)
				So(decision.Reason, ShouldEqual, "no entry signal")
			})
		})

		Convey("When max-position sizing is used", func() {
			decorated := &stubStrategy{decision: Decision{Action: ActionBuy, Reason: "signal"}}
			decorator := NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
				Decorated:           decorated,
				MaxPositionFraction: 0.1,
				StopLossPct:         0.02,
			})
			decision := decorator.Evaluate(EvaluateInput{Price: 100, BuyingPower: 1000})
			Convey("Then quantity is derived from max allocation", func() {
				So(decision.Action, ShouldEqual, ActionBuy)
				So(decision.Quantity, ShouldEqual, 1)
			})
		})

		Convey("When risk-per-trade sizing is used", func() {
			decorated := &stubStrategy{decision: Decision{Action: ActionBuy, Reason: "signal"}}
			decorator := NewPositionSizingDecorator(NewPositionSizingDecoratorInput{
				Decorated:           decorated,
				MaxPositionFraction: 0.5,
				RiskPerTradePct:     0.02,
				StopLossPct:         0.10,
			})
			decision := decorator.Evaluate(EvaluateInput{Price: 100, BuyingPower: 1000})
			Convey("Then quantity is capped by position fraction", func() {
				So(decision.Action, ShouldEqual, ActionBuy)
				So(decision.Quantity, ShouldEqual, 2)
			})
		})
	})
}
