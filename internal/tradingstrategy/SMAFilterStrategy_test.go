package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSMAFilterStrategy(t *testing.T) {
	Convey("Given an SMA filter strategy", t, func() {
		strategy := tradingstrategy.NewSMAFilterStrategy()

		Convey("When SMA is missing", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Price: 100})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "sma unavailable")
		})

		Convey("When price is below SMA", func() {
			sma := 110.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Price: 100, SMA: &sma})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "price below sma")
		})

		Convey("When price is above SMA", func() {
			sma := 90.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Price: 100, SMA: &sma})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})
}
