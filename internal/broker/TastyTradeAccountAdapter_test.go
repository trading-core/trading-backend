package broker_test

import (
	"context"
	"testing"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
	. "github.com/smartystreets/goconvey/convey"
)

type fakeTastyTradeClient struct {
	tastytrade.Client // embed so unused methods are present (and would panic if called)
	pagesByOffset map[int]*tastytrade.AccountTransactionsOutput
	calls         []tastytrade.GetAccountTransactionsInput
}

func (fake *fakeTastyTradeClient) GetAccountTransactions(ctx context.Context, input tastytrade.GetAccountTransactionsInput) (*tastytrade.AccountTransactionsOutput, error) {
	fake.calls = append(fake.calls, input)
	page, ok := fake.pagesByOffset[input.PageOffset]
	if !ok {
		return &tastytrade.AccountTransactionsOutput{}, nil
	}
	return page, nil
}

func TestTastyTradeAccountAdapterGetTransactions(t *testing.T) {
	Convey("Given a tastytrade client returning a single page of transactions", t, func() {
		fake := &fakeTastyTradeClient{
			pagesByOffset: map[int]*tastytrade.AccountTransactionsOutput{
				0: {
					Data: tastytrade.AccountTransactionsData{
						Items: []tastytrade.AccountTransaction{
							{
								ID:                  1,
								Symbol:              "AAPL",
								TransactionType:     "Trade",
								Action:              "Buy to Open",
								Quantity:            "10",
								Price:               "150.00",
								Value:               "1500.00",
								ValueEffect:         "Debit",
								Commission:          "1.00",
								CommissionEffect:    "Debit",
								RegulatoryFees:      "0.02",
								RegulatoryFeesEffect: "Debit",
								ClearingFees:        "0.10",
								ClearingFeesEffect:  "Debit",
								ExecutedAt:          "2026-04-20T14:30:00Z",
							},
							{
								ID:                  2,
								Symbol:              "AAPL",
								TransactionType:     "Trade",
								Action:              "Sell to Close",
								Quantity:            "10",
								Price:               "160.00",
								Value:               "100.00",
								ValueEffect:         "Credit",
								Commission:          "1.00",
								CommissionEffect:    "Debit",
								RegulatoryFees:      "0.03",
								RegulatoryFeesEffect: "Debit",
								ClearingFees:        "0.10",
								ClearingFeesEffect:  "Debit",
								ExecutedAt:          "2026-04-20T18:00:00Z",
							},
						},
					},
					Pagination: tastytrade.AccountTransactionsPagination{TotalPages: 1},
				},
			},
		}
		adapter := broker.NewTastyTradeAccountAdapter(broker.NewTastyTradeAccountAdapterInput{
			AccountID: "acct-1",
			Client:    fake,
		})

		Convey("When calling GetTransactions", func() {
			output, err := adapter.GetTransactions(context.Background(), broker.GetTransactionsInput{
				From: "2026-04-20",
				To:   "2026-04-20",
			})
			So(err, ShouldBeNil)

			Convey("Then transactions are returned", func() {
				So(len(output.Transactions), ShouldEqual, 2)
			})

			Convey("And the buy-to-open has action=buy, effect=open, signed debit value", func() {
				buy := output.Transactions[0]
				So(buy.Action, ShouldEqual, broker.OrderActionBuy)
				So(buy.Effect, ShouldEqual, broker.OrderEffectOpen)
				So(buy.Value, ShouldEqual, -1500.00)
				So(buy.RealizedPnL, ShouldEqual, 0)
			})

			Convey("And the sell-to-close has action=sell, effect=close; realized PnL is left for the aggregator", func() {
				sell := output.Transactions[1]
				So(sell.Action, ShouldEqual, broker.OrderActionSell)
				So(sell.Effect, ShouldEqual, broker.OrderEffectClose)
				So(sell.Value, ShouldEqual, 100.00)
				So(sell.RealizedPnL, ShouldEqual, 0)
			})

			Convey("And fees are summed in absolute terms", func() {
				buy := output.Transactions[0]
				So(buy.Fees, ShouldEqual, 1.12)
			})

			Convey("And request parameters are forwarded to the client", func() {
				So(fake.calls[0].AccountID, ShouldEqual, "acct-1")
				So(fake.calls[0].StartDate, ShouldEqual, "2026-04-20")
				So(fake.calls[0].EndDate, ShouldEqual, "2026-04-20")
			})
		})
	})

	Convey("Given a tastytrade client with multiple pages", t, func() {
		fake := &fakeTastyTradeClient{
			pagesByOffset: map[int]*tastytrade.AccountTransactionsOutput{
				0: {
					Data: tastytrade.AccountTransactionsData{
						Items: []tastytrade.AccountTransaction{
							{ID: 1, TransactionType: "Trade", Action: "Buy to Open", Quantity: "1", Price: "10", Value: "10", ValueEffect: "Debit", ExecutedAt: "2026-04-20T14:00:00Z"},
						},
					},
					Pagination: tastytrade.AccountTransactionsPagination{TotalPages: 2},
				},
				1: {
					Data: tastytrade.AccountTransactionsData{
						Items: []tastytrade.AccountTransaction{
							{ID: 2, TransactionType: "Trade", Action: "Sell to Close", Quantity: "1", Price: "12", Value: "2", ValueEffect: "Credit", ExecutedAt: "2026-04-20T15:00:00Z"},
						},
					},
					Pagination: tastytrade.AccountTransactionsPagination{TotalPages: 2},
				},
			},
		}
		adapter := broker.NewTastyTradeAccountAdapter(broker.NewTastyTradeAccountAdapterInput{
			AccountID: "acct-1",
			Client:    fake,
		})

		Convey("When calling GetTransactions", func() {
			output, err := adapter.GetTransactions(context.Background(), broker.GetTransactionsInput{})
			So(err, ShouldBeNil)

			Convey("Then transactions across all pages are returned", func() {
				So(len(output.Transactions), ShouldEqual, 2)
			})

			Convey("And the adapter paginated through both pages", func() {
				So(len(fake.calls), ShouldEqual, 2)
				So(fake.calls[0].PageOffset, ShouldEqual, 0)
				So(fake.calls[1].PageOffset, ShouldEqual, 1)
			})
		})
	})
}
