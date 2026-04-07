package tradingstrategy_test

import (
	"testing"
	"time"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestScalpingEvaluate(t *testing.T) {
	Convey("Given a scalping strategy with pullback parameters", t, func() {
		params := &tradingstrategy.PullbackParameters
		strategy := tradingstrategy.FromParameters(params)

		Convey("When there is an open order", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				HasOpenOrder: true,
				Price:        100,
				Now:          nyTimeForTest(11, 0),
			})
			Convey("Then it waits", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "waiting for open order to resolve")
			})
		})

		Convey("When there is an open position near session end", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				PositionQuantity: 7,
				Price:            100,
				EntryPrice:       100,
				Now:              nyTimeForTest(params.SessionEnd, 0),
			})
			Convey("Then exit decorator forces sell", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
				So(decision.Reason, ShouldEqual, "forced end-of-day exit")
				So(decision.Quantity, ShouldEqual, 7)
			})
		})

		Convey("When no buying power exists", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:       100,
				Now:         nyTimeForTest(11, 0),
				CashBalance: 0,
				BuyingPower: 0,
			})
			Convey("Then it does not enter", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "no buying power available")
			})
		})

		Convey("When pullback conditions are satisfied", func() {
			middle := 101.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:       100,
				BuyingPower: 1000,
				BollMiddle:  &middle,
				Now:         nyTimeForTest(11, 0),
			})
			// qty = floor(1000 * 0.25 / 100) = 2
			Convey("Then it buys with size derived from allocation", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldContainSubstring, "pullback")
				So(decision.Quantity, ShouldEqual, 2)
			})
		})
	})

	Convey("Given a scalping strategy with breakout parameters", t, func() {
		params := &tradingstrategy.BreakoutParameters
		strategy := tradingstrategy.FromParameters(params)

		Convey("When breakout conditions are satisfied", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:             102,   // above LookbackHighPrice (101)
				BuyingPower:       10000,
				SessionHighPrice:  100,
				LookbackHighPrice: 101,
				Now:               nyTimeForTest(11, 0),
			})
			Convey("Then it buys on breakout signal", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldContainSubstring, "breakout")
			})
		})
	})

	Convey("Given a scalping strategy with reentry cooldown", t, func() {
		params := &tradingstrategy.PullbackParameters
		strategy := tradingstrategy.FromParameters(params)

		Convey("When cooldown is active after stop loss", func() {
			now := nyTimeForTest(11, 0)
			lastStop := now.Add(-2 * time.Minute)
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:          100,
				BuyingPower:    1000,
				LastStopLossAt: &lastStop,
				Now:            now,
			})
			Convey("Then it prevents re-entry", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "re-entry cooldown active")
			})
		})
	})
}
