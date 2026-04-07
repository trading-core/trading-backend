package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBollingerFilterDecorator(t *testing.T) {
	Convey("Given a Bollinger filter decorator", t, func() {
		decorated := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"}}
		decorator := tradingstrategy.NewBollingerFilterDecorator(tradingstrategy.NewBollingerFilterDecoratorInput{
			Decorated:       decorated,
			RequireBreakout: true,
			MinWidthPct:     0.02,
			MaxWidthPct:     0.05,
		})

		Convey("When bollinger bands are missing", func() {
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100})
			So(decision.Reason, ShouldEqual, "bollinger unavailable")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When price is below upper bollinger", func() {
			upper := 101.0
			middle := 100.0
			lower := 99.0
			width := 0.03
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				Price:        100,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Reason, ShouldEqual, "price below upper bollinger")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When bollinger width is too narrow", func() {
			upper := 95.0
			middle := 100.0
			lower := 90.0
			width := 0.01
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				Price:        110,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Reason, ShouldEqual, "bollinger width too narrow")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When bollinger is not in squeeze", func() {
			upper := 95.0
			middle := 100.0
			lower := 90.0
			width := 0.06
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				Price:        110,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Reason, ShouldEqual, "bollinger not in squeeze")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When all bollinger filters pass", func() {
			upper := 95.0
			middle := 100.0
			lower := 90.0
			width := 0.03
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				Price:        110,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decorated.calls, ShouldEqual, 1)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})
	})
}
