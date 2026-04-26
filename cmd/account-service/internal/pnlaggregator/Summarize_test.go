package pnlaggregator_test

import (
	"testing"

	"github.com/kduong/trading-backend/cmd/account-service/internal/pnlaggregator"
	"github.com/kduong/trading-backend/internal/broker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSummarize(t *testing.T) {
	Convey("Given matched transactions with a mix of wins, losses and a scratch", t, func() {
		transactions := []broker.Transaction{
			{Type: "Trade", Effect: broker.OrderEffectOpen, RealizedPnL: 0, Fees: 1.0},
			{Type: "Trade", Effect: broker.OrderEffectClose, RealizedPnL: 200, Fees: 1.0},
			{Type: "Trade", Effect: broker.OrderEffectClose, RealizedPnL: -50, Fees: 1.0},
			{Type: "Trade", Effect: broker.OrderEffectClose, RealizedPnL: 0, Fees: 1.0},
			{Type: "Receive Deliver", RealizedPnL: 0, Fees: 0.10},
		}

		Convey("When summarizing", func() {
			summary := pnlaggregator.Summarize(transactions)

			Convey("Then only closing trades count toward TotalTrades", func() {
				So(summary.TotalTrades, ShouldEqual, 3)
			})

			Convey("And wins, losses, and scratches are partitioned correctly", func() {
				So(summary.WinningTrades, ShouldEqual, 1)
				So(summary.LosingTrades, ShouldEqual, 1)
			})

			Convey("And gross figures are signed sums of decided trades", func() {
				So(summary.GrossWins, ShouldEqual, 200.0)
				So(summary.GrossLosses, ShouldEqual, -50.0)
			})

			Convey("And NetPnL sums realized across closes only", func() {
				So(summary.NetPnL, ShouldEqual, 150.0)
			})

			Convey("And fees include every transaction in the window", func() {
				So(summary.Fees, ShouldEqual, 4.10)
			})

			Convey("And NetPnLAfterFees subtracts the full fee total", func() {
				So(summary.NetPnLAfterFees, ShouldEqual, 145.90)
			})

			Convey("And WinRate is wins / (wins + losses), excluding scratches", func() {
				So(summary.WinRate, ShouldEqual, 0.5)
			})
		})
	})

	Convey("Given no transactions", t, func() {
		summary := pnlaggregator.Summarize(nil)

		Convey("Then the summary is zeroed and WinRate is 0", func() {
			So(summary.TotalTrades, ShouldEqual, 0)
			So(summary.WinRate, ShouldEqual, 0.0)
		})
	})
}
