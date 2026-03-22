package broker

import (
	"context"

	"github.com/kduong/trading-backend/cmd/internal/account"
)

type AdapterFactory struct{}

func (factory *AdapterFactory) GetBrokerAdapter(ctx context.Context, object *account.Object) Adapter {
	switch object.BrokerType {
	case account.BrokerTypeTastyTrade:
		return NewTastyTradeAdapter()
	case account.BrokerTypeMockTest:
		return NewMockTestAdapter()
	default:
		panic("Unsupported broker type: " + object.BrokerType)
	}
}
