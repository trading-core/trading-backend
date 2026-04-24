package pnlaggregator_test

import (
	"testing"

	"github.com/kduong/trading-backend/cmd/account-service/internal/pnlaggregator"
	"github.com/kduong/trading-backend/internal/broker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAggregate(t *testing.T) {
	Convey("Given a set of transactions spanning multiple UTC days", t, func() {
		transactions := []broker.Transaction{
			{ExecutedAt: "2026-04-20T14:30:00Z", Type: "Trade", RealizedPnL: 100, Fees: 1.00},
			{ExecutedAt: "2026-04-20T18:00:00Z", Type: "Trade", RealizedPnL: 50, Fees: 0.50},
			{ExecutedAt: "2026-04-21T13:00:00Z", Type: "Trade", RealizedPnL: -30, Fees: 1.00},
			{ExecutedAt: "2026-04-22T09:00:00Z", Type: "Receive Deliver", RealizedPnL: 0, Fees: 0.10},
		}

		Convey("When aggregating", func() {
			result := pnlaggregator.Aggregate(transactions)

			Convey("Then there is one entry per calendar day that had activity", func() {
				So(len(result.Days), ShouldEqual, 3)
			})

			Convey("And days are sorted by date ascending", func() {
				So(result.Days[0].Date, ShouldEqual, "2026-04-20")
				So(result.Days[1].Date, ShouldEqual, "2026-04-21")
				So(result.Days[2].Date, ShouldEqual, "2026-04-22")
			})

			Convey("And realized PnL sums per day", func() {
				So(result.Days[0].RealizedPnL, ShouldEqual, 150.0)
				So(result.Days[1].RealizedPnL, ShouldEqual, -30.0)
			})

			Convey("And fees sum across all transaction types", func() {
				So(result.Days[0].Fees, ShouldEqual, 1.50)
				So(result.Days[2].Fees, ShouldEqual, 0.10)
			})

			Convey("And trade counts only include Trade-type transactions", func() {
				So(result.Days[0].TradeCount, ShouldEqual, 2)
				So(result.Days[1].TradeCount, ShouldEqual, 1)
				So(result.Days[2].TradeCount, ShouldEqual, 0)
			})
		})
	})

	Convey("Given transactions with UTC crossing midnight boundary", t, func() {
		transactions := []broker.Transaction{
			{ExecutedAt: "2026-04-20T23:59:00Z", Type: "Trade", RealizedPnL: 10},
			{ExecutedAt: "2026-04-21T00:01:00Z", Type: "Trade", RealizedPnL: 20},
		}

		Convey("When aggregating", func() {
			result := pnlaggregator.Aggregate(transactions)

			Convey("Then they land on separate UTC days", func() {
				So(len(result.Days), ShouldEqual, 2)
				So(result.Days[0].Date, ShouldEqual, "2026-04-20")
				So(result.Days[1].Date, ShouldEqual, "2026-04-21")
			})
		})
	})

	Convey("Given transactions with unparseable ExecutedAt", t, func() {
		transactions := []broker.Transaction{
			{ExecutedAt: "not-a-date", Type: "Trade", RealizedPnL: 10},
			{ExecutedAt: "2026-04-20T14:30:00Z", Type: "Trade", RealizedPnL: 5},
		}

		Convey("When aggregating", func() {
			result := pnlaggregator.Aggregate(transactions)

			Convey("Then unparseable rows are skipped without panicking", func() {
				So(len(result.Days), ShouldEqual, 1)
				So(result.Days[0].RealizedPnL, ShouldEqual, 5.0)
			})
		})
	})

	Convey("Given no transactions", t, func() {
		result := pnlaggregator.Aggregate(nil)

		Convey("Then the result has zero days", func() {
			So(len(result.Days), ShouldEqual, 0)
		})
	})
}
