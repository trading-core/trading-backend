package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBreakoutEntryStrategy(t *testing.T) {
	Convey("Given a breakout entry strategy with a 20-bar lookback and overbought RSI threshold of 70", t, func() {
		rsi := 60.0
		strategy := tradingstrategy.NewBreakoutEntryStrategy(tradingstrategy.NewBreakoutEntryStrategyInput{
			LookbackBars:  20,
			OverboughtRSI: 70,
		})

		fullInput := tradingstrategy.EvaluateInput{
			Price:             105,
			LookbackHighPrice: 100,
			PositionQuantity:  0,
			RSI:               &rsi,
		}

		Convey("When price exceeds the lookback high and RSI is below threshold", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
			So(decision.Reason, ShouldEqual, "breakout entry: price above lookback high")
		})

		Convey("When RSI is above the overbought threshold, entry is rejected", func() {
			input := fullInput
			overbought := 75.0
			input.RSI = &overbought
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi overbought at breakout")
		})

		Convey("When RSI equals the overbought threshold, entry is rejected", func() {
			input := fullInput
			atThreshold := 70.0
			input.RSI = &atThreshold
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi overbought at breakout")
		})

		Convey("When RSI is configured but data is missing, entry is rejected", func() {
			input := fullInput
			input.RSI = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi unavailable")
		})
	})

	Convey("Given a breakout entry strategy with a 20-bar lookback and no RSI check", t, func() {
		strategy := tradingstrategy.NewBreakoutEntryStrategy(tradingstrategy.NewBreakoutEntryStrategyInput{
			LookbackBars: 20,
		})

		fullInput := tradingstrategy.EvaluateInput{
			Price:             105,
			LookbackHighPrice: 100,
			PositionQuantity:  0,
		}

		Convey("When price exceeds the lookback high", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
			So(decision.Reason, ShouldEqual, "breakout entry: price above lookback high")
		})

		Convey("When price equals the lookback high, no breakout", func() {
			input := fullInput
			input.Price = 100
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "price not above lookback high")
		})

		Convey("When price is below the lookback high", func() {
			input := fullInput
			input.Price = 95
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "price not above lookback high")
		})

		Convey("When already in a position", func() {
			input := fullInput
			input.PositionQuantity = 5
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})

		Convey("When lookback high is unavailable (zero)", func() {
			input := fullInput
			input.LookbackHighPrice = 0
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "lookback high unavailable")
		})
	})

	Convey("Given a breakout entry strategy with LookbackBars=0 (disabled)", t, func() {
		strategy := tradingstrategy.NewBreakoutEntryStrategy(tradingstrategy.NewBreakoutEntryStrategyInput{
			LookbackBars: 0,
		})
		decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
			Price:             105,
			LookbackHighPrice: 100,
			PositionQuantity:  0,
		})
		Convey("Then it abstains", func() {
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "breakout entry disabled")
		})
	})
}
