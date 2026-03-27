package broker

import (
	"context"

	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type ClientFactory struct {
	TastyTradeClientFactory tastytrade.ClientFactory
}

func (factory *ClientFactory) GetClient(ctx context.Context, account *Account) Client {
	switch account.Type {
	case AccountTypeTastyTrade:
		return NewTastyTradeAdapter(NewTastyTradeAdapterInput{
			Account: account,
			Client:  factory.TastyTradeClientFactory.Create(),
		})
	default:
		panic("Unsupported broker type: " + account.Type)
	}
}
