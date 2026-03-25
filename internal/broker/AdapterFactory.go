package broker

import (
	"context"
	"net/url"

	"github.com/kduong/trading-backend/internal/auth"
)

type AdapterFactory struct {
	BrokerClientByType map[string]Client
}

type Client struct {
	APIURL       *url.URL
	TokenManager auth.TokenManager
}

func (factory *AdapterFactory) GetBrokerAdapter(ctx context.Context, account *Account) Adapter {
	brokerClient := factory.BrokerClientByType[account.Type]
	switch account.Type {
	case "tastytrade":
		return NewTastyTradeAdapter(NewTastyTradeAdapterInput{
			Account:        account,
			APIURL:         brokerClient.APIURL,
			GetAccessToken: brokerClient.TokenManager.GetAccessToken,
		})
	default:
		panic("Unsupported broker type: " + account.Type)
	}
}
