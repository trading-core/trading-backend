package broker

import (
	"context"

	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type TastyTradeAccountDiscoveryAdapter struct {
	client tastytrade.Client
}

type TastyTradeAccountDiscoveryAdapterInput struct {
	Client tastytrade.Client
}

func NewTastyTradeAccountDiscoveryAdapter(input TastyTradeAccountDiscoveryAdapterInput) *TastyTradeAccountDiscoveryAdapter {
	return &TastyTradeAccountDiscoveryAdapter{
		client: input.Client,
	}
}

func (client *TastyTradeAccountDiscoveryAdapter) ListAccountIDs(ctx context.Context) (accountIDs []string, err error) {
	accounts, err := client.client.ListAccounts(ctx)
	if err != nil {
		return
	}
	for _, payload := range accounts {
		for _, item := range payload.Data.Items {
			accountIDs = append(accountIDs, item.Account.AccountNumber)
		}
	}
	return
}
