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
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
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
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "no buying power available")
			})
		})

		Convey("When pullback conditions are satisfied", func() {
			rsi := 60.0
			macd := 2.0
			signal := 1.0
			middle := 101.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:       100,
				BuyingPower: 1000,
				RSI:         &rsi,
				MACD:        &macd,
				MACDSignal:  &signal,
				BollMiddle:  &middle,
				Now:         nyTimeForTest(11, 0),
			})
			Convey("Then it buys with size derived from allocation", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
				So(decision.Reason, ShouldEqual, "pullback")
				So(decision.Quantity, ShouldEqual, 1)
			})
		})
	})

	Convey("Given a scalping strategy with breakout parameters", t, func() {
		params := &tradingstrategy.BreakoutParameters
		strategy := tradingstrategy.FromParameters(params)

		Convey("When breakout conditions are satisfied", func() {
			rsi := 70.0
			macd := 2.0
			signal := 1.0
			bollUpper := 100.0
			bollMiddle := 99.0
			bollLower := 98.0
			bollWidth := 0.01 // 1% width
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:             102,   // Must be > LookbackHighPrice (101)
				BuyingPower:       10000, // Enough for at least 1 share at 102
				RSI:               &rsi,
				MACD:              &macd,
				MACDSignal:        &signal,
				BollUpper:         &bollUpper,
				BollMiddle:        &bollMiddle,
				BollLower:         &bollLower,
				BollWidthPct:      &bollWidth,
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
			rsi := 70.0
			macd := 2.0
			signal := 1.0
			middle := 101.0
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
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
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "re-entry cooldown active")
			})
		})
	})
}
