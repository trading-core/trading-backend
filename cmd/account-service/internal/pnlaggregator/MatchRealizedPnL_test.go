package pnlaggregator_test

import (
	"testing"

	"github.com/kduong/trading-backend/cmd/account-service/internal/pnlaggregator"
	"github.com/kduong/trading-backend/internal/broker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMatchRealizedPnL(t *testing.T) {
	Convey("Given a long round-trip on a single symbol (buy 10@100, sell 10@110)", t, func() {
		transactions := []broker.Transaction{
			{
				ExecutedAt: "2026-04-20T14:00:00Z",
				Type:       "Trade",
				Symbol:     "AAPL",
				Action:     broker.OrderActionBuy,
				Effect:     broker.OrderEffectOpen,
				Quantity:   10,
				Value:      -1000,
			},
			{
				ExecutedAt: "2026-04-20T18:00:00Z",
				Type:       "Trade",
				Symbol:     "AAPL",
				Action:     broker.OrderActionSell,
				Effect:     broker.OrderEffectClose,
				Quantity:   10,
				Value:      1100,
			},
		}

		Convey("When matching realized PnL", func() {
			pnlaggregator.MatchRealizedPnL(transactions)

			Convey("Then the open trade has zero realized PnL", func() {
				So(transactions[0].RealizedPnL, ShouldEqual, 0)
			})

			Convey("And the close trade realized PnL equals close + open cash", func() {
				So(transactions[1].RealizedPnL, ShouldEqual, 100.0)
			})
		})
	})

	Convey("Given a short round-trip (sell-to-open 10@100, buy-to-close 10@90)", t, func() {
		transactions := []broker.Transaction{
			{
				ExecutedAt: "2026-04-20T14:00:00Z",
				Type:       "Trade",
				Symbol:     "TSLA",
				Action:     broker.OrderActionSell,
				Effect:     broker.OrderEffectOpen,
				Quantity:   10,
				Value:      1000,
			},
			{
				ExecutedAt: "2026-04-20T18:00:00Z",
				Type:       "Trade",
				Symbol:     "TSLA",
				Action:     broker.OrderActionBuy,
				Effect:     broker.OrderEffectClose,
				Quantity:   10,
				Value:      -900,
			},
		}

		Convey("When matching realized PnL", func() {
			pnlaggregator.MatchRealizedPnL(transactions)

			Convey("Then the close has positive realized PnL of 100", func() {
				So(transactions[1].RealizedPnL, ShouldEqual, 100.0)
			})
		})
	})

	Convey("Given a partial close (buy 10@100, sell 4@120)", t, func() {
		transactions := []broker.Transaction{
			{
				ExecutedAt: "2026-04-20T14:00:00Z",
				Type:       "Trade",
				Symbol:     "MSFT",
				Action:     broker.OrderActionBuy,
				Effect:     broker.OrderEffectOpen,
				Quantity:   10,
				Value:      -1000,
			},
			{
				ExecutedAt: "2026-04-20T18:00:00Z",
				Type:       "Trade",
				Symbol:     "MSFT",
				Action:     broker.OrderActionSell,
				Effect:     broker.OrderEffectClose,
				Quantity:   4,
				Value:      480,
			},
		}

		Convey("When matching realized PnL", func() {
			pnlaggregator.MatchRealizedPnL(transactions)

			Convey("Then realized PnL is computed only on the matched 4 units", func() {
				So(transactions[1].RealizedPnL, ShouldEqual, 80.0)
			})
		})
	})

	Convey("Given FIFO matching across multiple opens at different prices", t, func() {
		transactions := []broker.Transaction{
			{ExecutedAt: "2026-04-20T10:00:00Z", Type: "Trade", Symbol: "NVDA", Action: broker.OrderActionBuy, Effect: broker.OrderEffectOpen, Quantity: 5, Value: -500},
			{ExecutedAt: "2026-04-20T11:00:00Z", Type: "Trade", Symbol: "NVDA", Action: broker.OrderActionBuy, Effect: broker.OrderEffectOpen, Quantity: 5, Value: -600},
			{ExecutedAt: "2026-04-20T12:00:00Z", Type: "Trade", Symbol: "NVDA", Action: broker.OrderActionSell, Effect: broker.OrderEffectClose, Quantity: 8, Value: 1040},
		}

		Convey("When matching realized PnL", func() {
			pnlaggregator.MatchRealizedPnL(transactions)

			Convey("Then the close consumes the first lot fully and 3 units from the second", func() {
				// Close unit value = 130. First lot unit = -100, matched 5 → (130-100)*5 = 150.
				// Second lot unit = -120, matched 3 → (130-120)*3 = 30.
				// Total = 180.
				So(transactions[2].RealizedPnL, ShouldEqual, 180.0)
			})
		})
	})

	Convey("Given a close with no matching open in the dataset", t, func() {
		transactions := []broker.Transaction{
			{
				ExecutedAt: "2026-04-20T14:00:00Z",
				Type:       "Trade",
				Symbol:     "GOOG",
				Action:     broker.OrderActionSell,
				Effect:     broker.OrderEffectClose,
				Quantity:   10,
				Value:      1100,
			},
		}

		Convey("When matching realized PnL", func() {
			pnlaggregator.MatchRealizedPnL(transactions)

			Convey("Then realized PnL falls back to the close-leg cash", func() {
				So(transactions[0].RealizedPnL, ShouldEqual, 1100.0)
			})
		})
	})

	Convey("Given opens for one symbol and a close for another", t, func() {
		transactions := []broker.Transaction{
			{ExecutedAt: "2026-04-20T10:00:00Z", Type: "Trade", Symbol: "AAPL", Action: broker.OrderActionBuy, Effect: broker.OrderEffectOpen, Quantity: 10, Value: -1000},
			{ExecutedAt: "2026-04-20T11:00:00Z", Type: "Trade", Symbol: "MSFT", Action: broker.OrderActionSell, Effect: broker.OrderEffectClose, Quantity: 5, Value: 600},
		}

		Convey("When matching realized PnL", func() {
			pnlaggregator.MatchRealizedPnL(transactions)

			Convey("Then the AAPL open does not match the MSFT close, which falls back to close cash", func() {
				So(transactions[0].RealizedPnL, ShouldEqual, 0)
				So(transactions[1].RealizedPnL, ShouldEqual, 600.0)
			})
		})
	})

	Convey("Given non-Trade transactions are present", t, func() {
		transactions := []broker.Transaction{
			{ExecutedAt: "2026-04-20T09:00:00Z", Type: "Receive Deliver", Symbol: "AAPL", Quantity: 10, Value: -1000},
			{ExecutedAt: "2026-04-20T10:00:00Z", Type: "Trade", Symbol: "AAPL", Action: broker.OrderActionBuy, Effect: broker.OrderEffectOpen, Quantity: 10, Value: -1000},
			{ExecutedAt: "2026-04-20T11:00:00Z", Type: "Trade", Symbol: "AAPL", Action: broker.OrderActionSell, Effect: broker.OrderEffectClose, Quantity: 10, Value: 1100},
		}

		Convey("When matching realized PnL", func() {
			pnlaggregator.MatchRealizedPnL(transactions)

			Convey("Then non-Trade rows are ignored and the round-trip resolves to 100", func() {
				So(transactions[2].RealizedPnL, ShouldEqual, 100.0)
			})
		})
	})
}
