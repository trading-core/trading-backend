package tradingstrategy_test

import (
	"testing"

	"github.com/kduong/trading-backend/internal/tradingstrategy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEntryGuardStrategy(t *testing.T) {
	Convey("Given an entry guard strategy", t, func() {
		strategy := tradingstrategy.NewEntryGuardStrategy()

		Convey("When flat", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{PositionQuantity: 0})
			Convey("Then it abstains", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionNone)
			})
		})

		Convey("When already in position", func() {
			decision := strategy.Evaluate(tradingstrategy.EvaluateInput{PositionQuantity: 10})
			Convey("Then it vetoes", func() {
				So(decision.Action, ShouldEqual, tradingstrategy.ActionVeto)
				So(decision.Reason, ShouldEqual, "already in position")
			})
		})
	})
}
