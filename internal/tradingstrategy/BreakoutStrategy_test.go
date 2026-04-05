package tradingstrategy

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBreakoutStrategy(t *testing.T) {
	Convey("Given a breakout strategy", t, func() {
		Convey("When lookback is one bar and price breaks session high", func() {
			strategy := NewBreakoutStrategy(NewBreakoutStrategyInput{LookbackBars: 1})
			decision := strategy.Evaluate(EvaluateInput{Price: 101, SessionHighPrice: 100})
			Convey("Then it emits a buy breakout decision", func() {
				So(decision.Action, ShouldEqual, ActionBuy)
				So(decision.Reason, ShouldEqual, "breakout above 1-bar high")
			})
		})

		Convey("When lookback is greater than one and lookback high is available", func() {
			strategy := NewBreakoutStrategy(NewBreakoutStrategyInput{LookbackBars: 5})
			decision := strategy.Evaluate(EvaluateInput{Price: 106, SessionHighPrice: 99, LookbackHighPrice: 105})
			Convey("Then it uses lookback high for breakout", func() {
				So(decision.Action, ShouldEqual, ActionBuy)
				So(decision.Reason, ShouldEqual, "breakout above 5-bar high")
			})
		})

		Convey("When breakout condition is not met", func() {
			strategy := NewBreakoutStrategy(NewBreakoutStrategyInput{LookbackBars: 3})
			decision := strategy.Evaluate(EvaluateInput{Price: 100, SessionHighPrice: 101, LookbackHighPrice: 0})
			Convey("Then it returns no action", func() {
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "no breakout")
			})
		})

		Convey("When reading strategy type", func() {
			strategy := NewBreakoutStrategy(NewBreakoutStrategyInput{LookbackBars: 1})
			So(strategy.Type(), ShouldEqual, StrategyTypeBreakoutTrading)
		})
	})
}
