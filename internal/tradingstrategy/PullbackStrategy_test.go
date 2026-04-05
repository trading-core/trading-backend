package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPullbackStrategy(t *testing.T) {
	Convey("Given a pullback strategy", t, func() {
		strategy := &tradingstrategy.PullbackStrategy{}

		Convey("When Bollinger middle is unavailable", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Price: 100})
			Convey("Then it returns no action with reason", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "bollinger middle unavailable for pullback")
			})
		})

		Convey("When price is at or below Bollinger middle", func() {
			middle := 101.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Price: 100, BollMiddle: &middle})
			Convey("Then it emits a buy decision", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldEqual, "pullback")
			})
		})

		Convey("When price is above Bollinger middle", func() {
			middle := 99.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Price: 100, BollMiddle: &middle})
			Convey("Then it does not enter", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

	})
}
