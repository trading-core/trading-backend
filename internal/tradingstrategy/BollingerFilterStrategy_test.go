package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBollingerFilterStrategy(t *testing.T) {
	Convey("Given a Bollinger filter strategy", t, func() {
		strategy := tradingstrategy.NewBollingerFilterStrategy(tradingstrategy.NewBollingerFilterStrategyInput{
			RequireBreakout: true,
			MinWidthPct:     0.02,
			MaxWidthPct:     0.05,
		})

		Convey("When bollinger bands are missing", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{Price: 100})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "bollinger unavailable")
		})

		Convey("When price is below upper bollinger", func() {
			upper := 101.0
			middle := 100.0
			lower := 99.0
			width := 0.03
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:        100,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "price below upper bollinger")
		})

		Convey("When bollinger width is too narrow", func() {
			upper := 95.0
			middle := 100.0
			lower := 90.0
			width := 0.01
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:        110,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "bollinger width too narrow")
		})

		Convey("When bollinger is not in squeeze", func() {
			upper := 95.0
			middle := 100.0
			lower := 90.0
			width := 0.06
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:        110,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
			So(decision.Reason, ShouldEqual, "bollinger not in squeeze")
		})

		Convey("When all bollinger filters pass", func() {
			upper := 95.0
			middle := 100.0
			lower := 90.0
			width := 0.03
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{
				Price:        110,
				BollUpper:    &upper,
				BollMiddle:   &middle,
				BollLower:    &lower,
				BollWidthPct: &width,
			})
			So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
		})
	})
}
