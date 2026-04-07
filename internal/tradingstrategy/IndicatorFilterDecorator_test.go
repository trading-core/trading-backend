package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSMAFilterDecorator(t *testing.T) {
	Convey("Given an SMA filter decorator", t, func() {
		decorated := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"}}
		decorator := tradingstrategy.NewSMAFilterDecorator(tradingstrategy.NewSMAFilterDecoratorInput{
			Decorated: decorated,
		})

		Convey("When SMA is missing", func() {
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100})
			So(decision.Reason, ShouldEqual, "sma unavailable")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When price is below SMA", func() {
			sma := 110.0
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100, SMA: &sma})
			So(decision.Reason, ShouldEqual, "price below sma")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When price is above SMA", func() {
			sma := 90.0
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100, SMA: &sma})
			So(decorated.calls, ShouldEqual, 1)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})
	})
}
