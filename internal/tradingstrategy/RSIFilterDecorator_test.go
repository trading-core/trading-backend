package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRSIFilterDecorator(t *testing.T) {
	Convey("Given an RSI filter decorator", t, func() {
		decorated := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"}}
		decorator := tradingstrategy.NewRSIFilterDecorator(tradingstrategy.NewRSIFilterDecoratorInput{
			Decorated: decorated,
			MinRSI:    40,
		})

		Convey("When RSI is missing", func() {
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100})
			So(decision.Reason, ShouldEqual, "rsi unavailable")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When RSI is below threshold", func() {
			rsi := 30.0
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100, RSI: &rsi})
			So(decision.Reason, ShouldEqual, "rsi below threshold")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When RSI meets threshold", func() {
			rsi := 55.0
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100, RSI: &rsi})
			So(decorated.calls, ShouldEqual, 1)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})
	})
}
