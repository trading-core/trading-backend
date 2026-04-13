package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTrendEntryStrategy(t *testing.T) {
	Convey("Given a trend entry strategy", t, func() {
		strategy := tradingstrategy.NewTrendEntryStrategy(tradingstrategy.NewTrendEntryStrategyInput{
			OverboughtRSI: 70,
		})

		macd := 2.0
		signal := 1.0
		sma := 90.0
		upper := 110.0
		rsi := 60.0

		fullInput := tradingstrategy.EvaluateInput{
			Price:            100,
			PositionQuantity: 0,
			MACD:             &macd,
			MACDSignal:       &signal,
			SMA:              &sma,
			BollUpper:        &upper,
			RSI:              &rsi,
		}

		Convey("When all conditions are met", func() {
			decision := strategy.Evaluate(fullInput)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})

		Convey("When MACD histogram minimum threshold is configured", func() {
			thresholdStrategy := tradingstrategy.NewTrendEntryStrategy(tradingstrategy.NewTrendEntryStrategyInput{
				OverboughtRSI:    70,
				MinMACDHistogram: 1.5,
			})

			Convey("And the histogram meets the threshold, entry is allowed", func() {
				// MACD=2.0, Signal=1.0 → histogram=1.0... wait need histogram >= 1.5
				// Use macd=2.5, signal=1.0 → histogram=1.5 exactly
				bigMACD := 2.5
				input := fullInput
				input.MACD = &bigMACD
				decision := thresholdStrategy.Evaluate(input)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
			})

			Convey("And the histogram is below the threshold, entry is rejected", func() {
				// MACD=2.0, Signal=1.0 → histogram=1.0 < threshold 1.5
				decision := thresholdStrategy.Evaluate(fullInput)
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
				So(decision.Reason, ShouldEqual, "macd histogram below minimum threshold")
			})
		})

		Convey("When MinMACDHistogram is zero (disabled), thin crossovers are allowed", func() {
			noThresholdStrategy := tradingstrategy.NewTrendEntryStrategy(tradingstrategy.NewTrendEntryStrategyInput{
				OverboughtRSI: 70,
			})
			thinMACD := 1.01
			thinSignal := 1.0
			input := fullInput
			input.MACD = &thinMACD
			input.MACDSignal = &thinSignal
			decision := noThresholdStrategy.Evaluate(input)
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
			noRSIStrategy := tradingstrategy.NewTrendEntryStrategy(tradingstrategy.NewTrendEntryStrategyInput{})
			input := fullInput
			input.RSI = nil
			decision := noRSIStrategy.Evaluate(input)
			So(decision.Action, ShouldEqual, tradingstrategy.ActionBuy)
		})
	})
}
