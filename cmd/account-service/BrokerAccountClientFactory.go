package main

import (
	"context"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type BrokerAccountClientFactory struct {
	TastyTradeClientFactory tastytrade.ClientFactory
}

func (factory *BrokerAccountClientFactory) Get(ctx context.Context, account *broker.Account) broker.AccountClient {
	switch account.Type {
	case broker.AccountTypeTastyTrade:
		return broker.NewTastyTradeAccountAdapter(broker.NewTastyTradeAccountAdapterInput{
			AccountID: account.ID,
			Client:    factory.TastyTradeClientFactory.Create(),
		})
	default:
		panic("Unsupported broker type: " + account.Type)
	}
}
