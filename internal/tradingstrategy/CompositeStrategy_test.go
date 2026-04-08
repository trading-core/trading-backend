package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCompositeStrategy(t *testing.T) {
	Convey("Given a composite strategy", t, func() {

		Convey("When the first strategy fires", func() {
			s1 := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "first"}}
			s2 := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "second"}}
			fm := tradingstrategy.NewCompositeStrategy(s1, s2)
			decision := fm.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then it returns immediately without evaluating the rest", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldEqual, "first")
				So(s2.calls, ShouldEqual, 0)
			})
		})

		Convey("When the first strategy abstains and the second fires", func() {
			s1 := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionNone}}
			s2 := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "second"}}
			fm := tradingstrategy.NewCompositeStrategy(s1, s2)
			decision := fm.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then it returns the second match", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldEqual, "second")
			})
		})

		Convey("When all strategies abstain", func() {
			fm := tradingstrategy.NewCompositeStrategy(
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionNone}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionNone}},
			)
			decision := fm.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then no action is taken", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("When a strategy vetoes", func() {
			s3 := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "third"}}
			fm := tradingstrategy.NewCompositeStrategy(
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionNone}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionVeto, Reason: "blocked"}},
				s3,
			)
			decision := fm.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then veto short-circuits and later strategies are not evaluated", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "blocked")
				So(s3.calls, ShouldEqual, 0)
			})
		})

	})
}
