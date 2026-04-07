package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMACDFilterDecorator(t *testing.T) {
	Convey("Given a MACD filter decorator", t, func() {
		decorated := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"}}
		decorator := tradingstrategy.NewMACDFilterDecorator(tradingstrategy.NewMACDFilterDecoratorInput{
			Decorated: decorated,
		})

		Convey("When MACD is missing", func() {
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100})
			So(decision.Reason, ShouldEqual, "macd unavailable")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When MACD signal is missing", func() {
			macd := 1.0
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100, MACD: &macd})
			So(decision.Reason, ShouldEqual, "macd unavailable")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When MACD is below signal", func() {
			macd := 1.0
			signal := 2.0
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100, MACD: &macd, MACDSignal: &signal})
			So(decision.Reason, ShouldEqual, "macd below signal")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When MACD is above signal", func() {
			macd := 3.0
			signal := 2.0
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100, MACD: &macd, MACDSignal: &signal})
			So(decorated.calls, ShouldEqual, 1)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})
	})
}
