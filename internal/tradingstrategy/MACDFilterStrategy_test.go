package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMACDFilterStrategy(t *testing.T) {
	Convey("Given a MACD filter strategy", t, func() {
		strategy := tradingstrategy.NewMACDFilterStrategy()

		Convey("When MACD is missing", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "macd unavailable")
		})

		Convey("When MACD signal is missing", func() {
			macd := 1.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{MACD: &macd})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "macd unavailable")
		})

		Convey("When MACD is below signal", func() {
			macd := 1.0
			signal := 2.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{MACD: &macd, MACDSignal: &signal})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "macd below signal")
		})

		Convey("When MACD is above signal", func() {
			macd := 3.0
			signal := 2.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{MACD: &macd, MACDSignal: &signal})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})
}
