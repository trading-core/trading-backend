package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTrendEntryStrategy(t *testing.T) {
	Convey("Given a trend entry strategy", t, func() {
		strategy := tradingstrategy.NewTrendEntryStrategy(tradingstrategy.NewTrendEntryStrategyInput{
			MinBollingerWidthPct: 0.02,
			MaxBollingerWidthPct: 0.10,
			OverboughtRSI:        70,
		})

		macd := 2.0
		signal := 1.0
		sma := 90.0
		upper := 110.0
		width := 0.05
		rsi := 60.0

		fullInput := tradingstrategy.EvaluateInput{
			Price:            100,
			PositionQuantity: 0,
			MACD:             &macd,
			MACDSignal:       &signal,
			SMA:              &sma,
			BollUpper:        &upper,
			BollWidthPct:     &width,
			RSI:              &rsi,
		}

		Convey("When all conditions are met", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})

		Convey("When MACD is not above signal", func() {
			input := fullInput
			low := 0.5
			input.MACD = &low
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "macd not above signal")
		})

		Convey("When price is not above SMA", func() {
			input := fullInput
			highSMA := 105.0
			input.SMA = &highSMA
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "price not above sma")
		})

		Convey("When price is at or above upper Bollinger, entry is rejected", func() {
			input := fullInput
			lowUpper := 95.0
			input.BollUpper = &lowUpper
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "price at or above upper bollinger")
		})

		Convey("When Bollinger width is too narrow", func() {
			input := fullInput
			narrow := 0.01
			input.BollWidthPct = &narrow
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "bollinger width too narrow")
		})

		Convey("When Bollinger width is too wide", func() {
			input := fullInput
			wide := 0.15
			input.BollWidthPct = &wide
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "bollinger not in squeeze")
		})

		Convey("When MACD data is missing", func() {
			input := fullInput
			input.MACD = nil
			decision := strategy.Evaluate(input)
			// Missing data → ActionNone so downstream strategies can still run
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "macd unavailable")
		})

		Convey("When Bollinger upper is missing, entry is still allowed", func() {
			input := fullInput
			input.BollUpper = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})

		Convey("When RSI is overbought, entry is rejected", func() {
			input := fullInput
			overbought := 75.0
			input.RSI = &overbought
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi overbought")
		})

		Convey("When RSI equals the overbought threshold, entry is rejected", func() {
			input := fullInput
			atThreshold := 70.0
			input.RSI = &atThreshold
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi overbought")
		})

		Convey("When RSI is configured but data is missing, entry is rejected", func() {
			input := fullInput
			input.RSI = nil
			decision := strategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			So(decision.Reason, ShouldEqual, "rsi unavailable")
		})

		Convey("When RSI check is disabled (zero threshold), missing RSI still allows entry", func() {
			noRSIStrategy := tradingstrategy.NewTrendEntryStrategy(tradingstrategy.NewTrendEntryStrategyInput{
				MinBollingerWidthPct: 0.02,
				MaxBollingerWidthPct: 0.10,
			})
			input := fullInput
			input.RSI = nil
			decision := noRSIStrategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})
	})
}
