package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestOverboughtExitStrategy(t *testing.T) {
	Convey("Given an overbought exit strategy with RSI threshold 70", t, func() {
		overboughtRSI := 75.0
		normalRSI := 60.0
		bollUpper := 100.0 // price is at the upper band
		strategy := tradingstrategy.NewOverboughtExitStrategy(tradingstrategy.NewOverboughtExitStrategyInput{
			OverboughtRSI: 70,
		})
		fullInput := tradingstrategy.EvaluateInput{
			Price:            100,
			PositionQuantity: 10,
			RSI:              &overboughtRSI,
			BollUpper:        &bollUpper,
		}

		Convey("When RSI is overbought and price is at the upper Bollinger Band, exit fires", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
			So(decision.Reason, ShouldEqual, "overbought exit: rsi overbought and price at upper bollinger")
			So(decision.Quantity, ShouldEqual, 10)
		})

		Convey("When RSI is overbought and price is above the lookback high, hold (genuine breakout)", func() {
			lookbackHigh := 95.0
			input := fullInput
			input.LookbackHighPrice = lookbackHigh // price 100 > lookback 95
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldContainSubstring, "breaking out above lookback high")
		})

		Convey("When RSI is overbought and price equals the lookback high, exit fires (not a new high)", func() {
			input := fullInput
			input.LookbackHighPrice = fullInput.Price
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
		})

		Convey("When RSI is overbought and lookback high is zero (disabled), exit fires", func() {
			input := fullInput
			input.LookbackHighPrice = 0
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionSell)
		})

		Convey("When RSI is overbought but price is below the upper Bollinger Band, exit does not fire", func() {
			highBoll := 110.0
			input := fullInput
			input.BollUpper = &highBoll
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi overbought but price below upper bollinger")
		})

		Convey("When RSI is overbought but Bollinger upper is missing, exit does not fire", func() {
			input := fullInput
			input.BollUpper = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "boll_upper unavailable")
		})

		Convey("When RSI is not overbought, exit does not fire", func() {
			input := fullInput
			input.RSI = &normalRSI
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi not overbought")
		})

		Convey("When RSI data is missing, exit does not fire", func() {
			input := fullInput
			input.RSI = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi unavailable")
		})

		Convey("When not holding a position", func() {
			input := fullInput
			input.PositionQuantity = 0
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})

	Convey("Given an overbought exit strategy with OverboughtRSI=0 (disabled)", t, func() {
		strategy := tradingstrategy.NewOverboughtExitStrategy(tradingstrategy.NewOverboughtExitStrategyInput{})
		rsi := 80.0
		decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
			Price:            100,
			PositionQuantity: 10,
			RSI:              &rsi,
		})
		Convey("Then it abstains", func() {
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})
}
