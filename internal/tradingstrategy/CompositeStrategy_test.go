package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCompositeStrategy(t *testing.T) {
	Convey("Given a composite strategy", t, func() {

		Convey("When a single strategy is composed", func() {
			s := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"}}
			composite := tradingstrategy.NewCompositeStrategy(s)
			decision := composite.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then it passes through the decision", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldContainSubstring, "signal")
			})
		})

		Convey("When all strategies agree on Buy", func() {
			composite := tradingstrategy.NewCompositeStrategy(
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "a"}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "b"}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "c"}},
			)
			decision := composite.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then Buy wins unanimously", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldEqual, "a; b; c")
			})
		})

		Convey("When the majority vote for Buy", func() {
			composite := tradingstrategy.NewCompositeStrategy(
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "a"}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "b"}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionNone, Reason: "c"}},
			)
			decision := composite.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then Buy wins by plurality", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldEqual, "a; b")
			})
		})

		Convey("When the majority vote for Sell", func() {
			composite := tradingstrategy.NewCompositeStrategy(
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionSell, Reason: "x"}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionSell, Reason: "y"}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "z"}},
			)
			decision := composite.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then Sell wins by plurality", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldEqual, "x; y")
			})
		})

		Convey("When Buy and Sell are tied", func() {
			composite := tradingstrategy.NewCompositeStrategy(
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "a"}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionSell, Reason: "b"}},
			)
			decision := composite.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then no action is taken", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("When all strategies return None", func() {
			composite := tradingstrategy.NewCompositeStrategy(
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionNone}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionNone}},
			)
			decision := composite.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then no action is taken", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("When one strategy returns Veto", func() {
			third := &stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "signal"}}
			composite := tradingstrategy.NewCompositeStrategy(
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "a"}},
				&stubStrategy{decision: tradingstrategy.Decision{Action: tradingstrategy.ActionVeto, Reason: "blocked"}},
				third,
			)
			decision := composite.Evaluate(tradingstrategy.EvaluateInput{})
			Convey("Then Veto overrides all votes and short-circuits", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "blocked")
				So(third.calls, ShouldEqual, 0)
			})
		})

	})
}
