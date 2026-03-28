package broker

import (
	"context"

	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type AccountClientFactory struct {
	TastyTradeClientFactory tastytrade.ClientFactory
}

func (factory *AccountClientFactory) Get(ctx context.Context, account *Account) AccountClient {
	switch account.Type {
	case AccountTypeTastyTrade:
		return NewTastyTradeAdapter(NewTastyTradeAdapterInput{
			AccountID: account.TastyTrade.ID,
			Client:    factory.TastyTradeClientFactory.Create(),
		})
	default:
		panic("Unsupported broker type: " + account.Type)
	}
}
