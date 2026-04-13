package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

type stubRegimeDetector struct {
	regime tradingstrategy.Regime
}

func (detector *stubRegimeDetector) Detect(_ tradingstrategy.EvaluateInput) tradingstrategy.Regime {
	return detector.regime
}

func TestRegimeSwitchStrategy(t *testing.T) {
	uptrendDecision := tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "uptrend entry"}
	rangeDecision := tradingstrategy.Decision{Action: tradingstrategy.ActionBuy, Reason: "range entry"}
	downtrendDecision := tradingstrategy.Decision{Action: tradingstrategy.ActionNone, Reason: "noop"}

	uptrendStub := &stubStrategy{decision: uptrendDecision}
	rangeStub := &stubStrategy{decision: rangeDecision}
	downtrendStub := &stubStrategy{decision: downtrendDecision}

	detector := &stubRegimeDetector{}

	strategy := tradingstrategy.NewRegimeSwitchStrategy(tradingstrategy.NewRegimeSwitchStrategyInput{
		Detector:  detector,
		Uptrend:   uptrendStub,
		Range:     rangeStub,
		Downtrend: downtrendStub,
	})

	input := tradingstrategy.EvaluateInput{Price: 100}

	Convey("Given a regime switch strategy", t, func() {
		uptrendStub.calls = 0
		rangeStub.calls = 0
		downtrendStub.calls = 0

		Convey("When the detector returns RegimeUptrend", func() {
			detector.regime = tradingstrategy.RegimeUptrend
			decision := strategy.Evaluate(input)

			Convey("Then it delegates to the uptrend strategy", func() {
				So(decision, ShouldResemble, uptrendDecision)
				So(uptrendStub.calls, ShouldEqual, 1)
				So(rangeStub.calls, ShouldEqual, 0)
				So(downtrendStub.calls, ShouldEqual, 0)
			})
		})

		Convey("When the detector returns RegimeRange", func() {
			detector.regime = tradingstrategy.RegimeRange
			decision := strategy.Evaluate(input)

			Convey("Then it delegates to the range strategy", func() {
				So(decision, ShouldResemble, rangeDecision)
				So(uptrendStub.calls, ShouldEqual, 0)
				So(rangeStub.calls, ShouldEqual, 1)
				So(downtrendStub.calls, ShouldEqual, 0)
			})
		})

		Convey("When the detector returns RegimeDowntrend", func() {
			detector.regime = tradingstrategy.RegimeDowntrend
			decision := strategy.Evaluate(input)

			Convey("Then it delegates to the downtrend strategy", func() {
				So(decision, ShouldResemble, downtrendDecision)
				So(uptrendStub.calls, ShouldEqual, 0)
				So(rangeStub.calls, ShouldEqual, 0)
				So(downtrendStub.calls, ShouldEqual, 1)
			})
		})
	})
}
