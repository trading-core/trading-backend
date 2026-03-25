package broker

import (
	"context"
	"sync"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
)

type AdapterFactory struct {
	mutex                    sync.Mutex
	brokerCredentialsByType  map[string]auth.Credentials
	tokenManagerByBrokerType map[string]auth.TokenManager
}

type NewAdapterFactoryInput struct {
	BrokerCredentialsByType map[string]auth.Credentials
}

func NewAdapterFactory(input NewAdapterFactoryInput) *AdapterFactory {
	return &AdapterFactory{
		brokerCredentialsByType:  input.BrokerCredentialsByType,
		tokenManagerByBrokerType: make(map[string]auth.TokenManager),
	}
}

func (factory *AdapterFactory) GetBrokerAdapter(ctx context.Context, account *account.Account) Adapter {
	switch account.BrokerType {
	case broker.TypeTastyTrade:
		credentials := factory.brokerCredentialsByType[account.BrokerType]
		tokenManager := factory.getOrCreateTokenManager(account.BrokerType, credentials.AuthorizationServer)
		return NewTastyTradeAdapter(NewTastyTradeAdapterInput{
			Account:        account,
			RawAPIURL:      credentials.APIURL,
			GetAccessToken: tokenManager.GetAccessToken,
		})
	default:
		panic("Unsupported broker type: " + account.BrokerType)
	}
}

func (factory *AdapterFactory) getOrCreateTokenManager(brokerType string, authorizationServerInfo auth.AuthorizationServerInfo) auth.TokenManager {
	factory.mutex.Lock()
	defer factory.mutex.Unlock()
	tokenManager, ok := factory.tokenManagerByBrokerType[brokerType]
	if ok {
		return tokenManager
	}
	switch brokerType {
	case broker.TypeTastyTrade:
		tokenManager = auth.NewTastyTradeTokenManager(&authorizationServerInfo)
	default:
		panic("Unsupported broker type: " + brokerType)
	}
	factory.tokenManagerByBrokerType[brokerType] = tokenManager
	return tokenManager
}
