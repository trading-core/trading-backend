package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEMABasedRegimeDetector(t *testing.T) {
	fastEMA := 110.0
	slowEMA := 100.0
	strongADX := 25.0
	weakADX := 15.0

	Convey("Given an EMA-based regime detector with ADX filtering enabled", t, func() {
		detector := tradingstrategy.NewEMABasedRegimeDetector(tradingstrategy.NewEMABasedRegimeDetectorInput{
			ADXThreshold: 20,
		})

		Convey("When FastEMA > SlowEMA and ADX is above threshold", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &fastEMA,
				SlowEMA: &slowEMA,
				ADX:     &strongADX,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeUptrend)
		})

		Convey("When FastEMA < SlowEMA and ADX is above threshold", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &slowEMA,
				SlowEMA: &fastEMA,
				ADX:     &strongADX,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeDowntrend)
		})

		Convey("When FastEMA > SlowEMA but ADX is below threshold", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &fastEMA,
				SlowEMA: &slowEMA,
				ADX:     &weakADX,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeRange)
		})

		Convey("When FastEMA > SlowEMA but ADX is unavailable", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &fastEMA,
				SlowEMA: &slowEMA,
				ADX:     nil,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeRange)
		})

		Convey("When FastEMA is unavailable", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: nil,
				SlowEMA: &slowEMA,
				ADX:     &strongADX,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeRange)
		})

		Convey("When SlowEMA is unavailable", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &fastEMA,
				SlowEMA: nil,
				ADX:     &strongADX,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeRange)
		})
	})

	Convey("Given an EMA-based regime detector with ADX filtering disabled", t, func() {
		detector := tradingstrategy.NewEMABasedRegimeDetector(tradingstrategy.NewEMABasedRegimeDetectorInput{
			ADXThreshold: 0,
		})

		Convey("When FastEMA > SlowEMA, it detects uptrend regardless of ADX", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &fastEMA,
				SlowEMA: &slowEMA,
				ADX:     nil,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeUptrend)
		})

		Convey("When FastEMA < SlowEMA, it detects downtrend regardless of ADX", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &slowEMA,
				SlowEMA: &fastEMA,
				ADX:     nil,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeDowntrend)
		})

		Convey("When both EMAs are unavailable, it returns range", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: nil,
				SlowEMA: nil,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeRange)
		})
	})

	Convey("Given an EMA-based regime detector with EMA separation hysteresis", t, func() {
		// FastEMA=110, SlowEMA=100 → separation = (110-100)/100 = 10%
		// FastEMA=101, SlowEMA=100 → separation = (101-100)/100 = 1%
		narrowFastEMA := 101.0
		detector := tradingstrategy.NewEMABasedRegimeDetector(tradingstrategy.NewEMABasedRegimeDetectorInput{
			EMASeparationThreshold: 0.05, // 5% required
		})

		Convey("When EMA separation exceeds the threshold, uptrend is confirmed", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &fastEMA, // 110 vs 100 = 10% separation
				SlowEMA: &slowEMA,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeUptrend)
		})

		Convey("When EMA separation is below the threshold, range is returned (hysteresis)", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &narrowFastEMA, // 101 vs 100 = 1% separation
				SlowEMA: &slowEMA,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeRange)
		})

		Convey("When FastEMA < SlowEMA and separation exceeds threshold, downtrend is confirmed", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &slowEMA,   // 100 vs 110 = −9.1% separation
				SlowEMA: &fastEMA,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeDowntrend)
		})

		Convey("When FastEMA < SlowEMA but separation is below threshold, range is returned", func() {
			input := tradingstrategy.EvaluateInput{
				FastEMA: &slowEMA,       // 100 vs 101 = −1% separation
				SlowEMA: &narrowFastEMA,
			}
			So(detector.Detect(input), ShouldEqual, tradingstrategy.RegimeRange)
		})
	})
}
