package tradingstrategy

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScalpingEvaluate(t *testing.T) {
	Convey("Given a scalping strategy", t, func() {
		strategy := NewScalping()

		Convey("When there is an open order", func() {
			decision := strategy.Evaluate(EvaluateInput{
				HasOpenOrder: true,
				Price:        100,
				Now:          nyTimeForTest(11, 0),
			})
			Convey("Then it waits", func() {
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "waiting for open order to resolve")
			})
		})

		Convey("When there is an open position near session end", func() {
			decision := strategy.Evaluate(EvaluateInput{
				PositionQuantity: 7,
				Price:            100,
				EntryPrice:       100,
				Now:              nyTimeForTest(strategy.SessionEnd, 0),
			})
			Convey("Then exit decorator forces sell", func() {
				So(decision.Action, ShouldEqual, ActionSell)
				So(decision.Reason, ShouldEqual, "forced end-of-day exit")
				So(decision.Quantity, ShouldEqual, 7)
			})
		})

		Convey("When no buying power exists", func() {
			decision := strategy.Evaluate(EvaluateInput{
				Price:       100,
				Now:         nyTimeForTest(11, 0),
				CashBalance: 0,
				BuyingPower: 0,
			})
			Convey("Then it does not enter", func() {
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "no buying power available")
			})
		})

		Convey("When pullback conditions are satisfied", func() {
			rsi := 60.0
			macd := 2.0
			signal := 1.0
			middle := 101.0
			decision := strategy.Evaluate(EvaluateInput{
				Price:       100,
				BuyingPower: 1000,
				RSI:         &rsi,
				MACD:        &macd,
				MACDSignal:  &signal,
				BollMiddle:  &middle,
				Now:         nyTimeForTest(11, 0),
			})
			Convey("Then it buys with size derived from allocation", func() {
				So(decision.Action, ShouldEqual, ActionBuy)
				So(decision.Reason, ShouldEqual, "entry signal: pullback")
				So(decision.Quantity, ShouldEqual, 1)
			})
		})

		Convey("When breakout mode is configured and breakout occurs", func() {
			strategy.EntryMode = "breakout"
			strategy.BreakoutLookbackBars = 5
			rsi := 70.0
			macd := 2.0
			signal := 1.0
			decision := strategy.Evaluate(EvaluateInput{
				Price:             100,
				BuyingPower:       1000,
				RSI:               &rsi,
				MACD:              &macd,
				MACDSignal:        &signal,
				SessionHighPrice:  90,
				LookbackHighPrice: 99,
				Now:               nyTimeForTest(11, 0),
			})
			Convey("Then it buys on breakout signal", func() {
				So(decision.Action, ShouldEqual, ActionBuy)
				So(decision.Reason, ShouldEqual, "entry signal: breakout")
			})
		})

		Convey("When cooldown is active after stop loss", func() {
			now := nyTimeForTest(11, 0)
			lastStop := now.Add(-2 * time.Minute)
			rsi := 70.0
			macd := 2.0
			signal := 1.0
			middle := 101.0
			decision := strategy.Evaluate(EvaluateInput{
				Price:          100,
				BuyingPower:    1000,
				RSI:            &rsi,
				MACD:           &macd,
				MACDSignal:     &signal,
				BollMiddle:     &middle,
				LastStopLossAt: &lastStop,
				Now:            now,
			})
			Convey("Then it prevents re-entry", func() {
				So(decision.Action, ShouldEqual, ActionNone)
				So(decision.Reason, ShouldEqual, "re-entry cooldown active")
			})
		})
	})
}

func TestScalpingType(t *testing.T) {
	Convey("Given a scalping strategy", t, func() {
		strategy := NewScalping()
		Convey("When requesting type", func() {
			So(strategy.Type(), ShouldEqual, StrategyTypeScalping)
		})
	})
}
