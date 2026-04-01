package main

import (
	"context"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type BrokerMarketDataClientFactory struct {
	TastyTradeClientFactory        tastytrade.ClientFactory
	TastyTradeSandboxClientFactory tastytrade.ClientFactory
}

func (factory *BrokerMarketDataClientFactory) Get(ctx context.Context, account *broker.Account) broker.MarketDataClient {
	switch account.Type {
	case broker.AccountTypeTastyTrade:
		return broker.NewTastyTradeMarketDataAdapter(broker.NewTastyTradeMarketDataAdapterInput{
			Client: factory.TastyTradeClientFactory.Create(),
		})
	case broker.AccountTypeTastyTradeSandbox:
		return broker.NewTastyTradeMarketDataAdapter(broker.NewTastyTradeMarketDataAdapterInput{
			Client: factory.TastyTradeSandboxClientFactory.Create(),
		})
	default:
		panic("Unsupported broker type: " + account.Type)
	}
}
