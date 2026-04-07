package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRSIFilterStrategy(t *testing.T) {
	Convey("Given an RSI filter strategy with minRSI (momentum mode)", t, func() {
		strategy := tradingstrategy.NewRSIFilterStrategy(tradingstrategy.NewRSIFilterStrategyInput{MinRSI: 40})

		Convey("When RSI is missing", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "rsi unavailable")
		})

		Convey("When RSI is below threshold", func() {
			rsi := 30.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{RSI: &rsi})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "rsi below threshold")
		})

		Convey("When RSI meets threshold", func() {
			rsi := 55.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{RSI: &rsi})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})

	Convey("Given an RSI filter strategy with maxRSI (mean-reversion mode)", t, func() {
		strategy := tradingstrategy.NewRSIFilterStrategy(tradingstrategy.NewRSIFilterStrategyInput{MaxRSI: 30})

		Convey("When RSI is above threshold (not oversold)", func() {
			rsi := 55.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{RSI: &rsi})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "rsi above threshold")
		})

		Convey("When RSI is in oversold territory", func() {
			rsi := 25.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{RSI: &rsi})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})
}
