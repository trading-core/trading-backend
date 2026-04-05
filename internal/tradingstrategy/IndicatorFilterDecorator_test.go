package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIndicatorFilterDecorator(t *testing.T) {
	Convey("Given an indicator filter decorator", t, func() {
		decorated := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"}}
		decorator := tradingstrategy.NewIndicatorFilterDecorator(tradingstrategy.NewIndicatorFilterDecoratorInput{
			Decorated:                decorated,
			MinRSI:                   40,
			RequireMACDSignal:        true,
			RequireBollingerBreakout: true,
			MinBollingerWidthPct:     0.02,
			MaxBollingerWidthPct:     0.05,
		})

		Convey("When RSI is missing", func() {
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100})
			So(decision.Reason, ShouldEqual, "rsi unavailable")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When MACD is below signal", func() {
			rsi := 55.0
			macd := 1.0
			signal := 2.0
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{Price: 100, RSI: &rsi, MACD: &macd, MACDSignal: &signal})
			So(decision.Reason, ShouldEqual, "macd below signal")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When bollinger breakout precondition fails", func() {
			rsi := 55.0
			macd := 3.0
			signal := 2.0
			upper := 101.0
			middle := 100.0
			lower := 99.0
			width := 0.03
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				Price:        100,
				RSI:          &rsi,
				MACD:         &macd,
				MACDSignal:   &signal,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Reason, ShouldEqual, "price below upper bollinger")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When squeeze requirement fails", func() {
			rsi := 55.0
			macd := 3.0
			signal := 2.0
			upper := 95.0
			middle := 100.0
			lower := 90.0
			width := 0.06
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				Price:        110,
				RSI:          &rsi,
				MACD:         &macd,
				MACDSignal:   &signal,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Reason, ShouldEqual, "bollinger not in squeeze")
			So(decorated.calls, ShouldEqual, 0)
		})

		Convey("When all filters pass", func() {
			rsi := 55.0
			macd := 3.0
			signal := 2.0
			upper := 95.0
			middle := 100.0
			lower := 90.0
			width := 0.03
			decision := decorator.Evaluate(tradingstrategy.EvaluateInput{
				Price:        110,
				RSI:          &rsi,
				MACD:         &macd,
				MACDSignal:   &signal,
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
