package pnlaggregator_test

import (
	"testing"

	"github.com/kduong/trading-backend/cmd/account-service/internal/pnlaggregator"
	"github.com/kduong/trading-backend/internal/broker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFilterByDateRange(t *testing.T) {
	Convey("Given transactions spanning before, during, and after the requested window", t, func() {
		transactions := []broker.Transaction{
			{ID: "before", ExecutedAt: "2026-03-15T14:00:00Z"},
			{ID: "from-edge", ExecutedAt: "2026-04-01T00:00:00Z"},
			{ID: "middle", ExecutedAt: "2026-04-15T18:00:00Z"},
			{ID: "to-edge", ExecutedAt: "2026-04-30T23:59:00Z"},
			{ID: "after", ExecutedAt: "2026-05-01T00:00:00Z"},
			{ID: "bad-date", ExecutedAt: "not-a-date"},
		}

		Convey("When filtering to [2026-04-01, 2026-04-30]", func() {
			result := pnlaggregator.FilterByDateRange(transactions, "2026-04-01", "2026-04-30")

			Convey("Then only rows whose UTC date falls within the inclusive window are kept", func() {
				So(len(result), ShouldEqual, 3)
				So(result[0].ID, ShouldEqual, "from-edge")
				So(result[1].ID, ShouldEqual, "middle")
				So(result[2].ID, ShouldEqual, "to-edge")
			})
		})
	})

	Convey("Given an empty slice", t, func() {
		result := pnlaggregator.FilterByDateRange(nil, "2026-04-01", "2026-04-30")

		Convey("Then the result is empty", func() {
			So(len(result), ShouldEqual, 0)
		})
	})
}
