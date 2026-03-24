package broker

import (
	"context"
	"sync"

	"github.com/kduong/trading-backend/cmd/internal/account"
)

type AdapterFactory struct {
	mutex                    sync.Mutex
	brokerCredentialsByType  map[account.BrokerType]Credentials
	tokenManagerByBrokerType map[account.BrokerType]TokenManager
}

type NewAdapterFactoryInput struct {
	BrokerCredentialsByType map[account.BrokerType]Credentials
}

func NewAdapterFactory(input NewAdapterFactoryInput) *AdapterFactory {
	return &AdapterFactory{
		brokerCredentialsByType:  input.BrokerCredentialsByType,
		tokenManagerByBrokerType: make(map[account.BrokerType]TokenManager),
	}
}

func (factory *AdapterFactory) GetBrokerAdapter(ctx context.Context, object *account.Object) Adapter {
	switch object.BrokerType {
	case account.BrokerTypeTastyTrade:
		credentials := factory.brokerCredentialsByType[object.BrokerType]
		tokenManager := factory.getOrCreateTokenManager(object.BrokerType, credentials.AuthorizationServer)
		return NewTastyTradeAdapter(NewTastyTradeAdapterInput{
			AccountObject:  object,
			APIEndpoint:    credentials.APIURL,
			GetAccessToken: tokenManager.GetAccessToken,
		})
	case account.BrokerTypeMockTest:
		return NewMockTestAdapter()
	default:
		panic("Unsupported broker type: " + object.BrokerType)
	}
}

func (factory *AdapterFactory) getOrCreateTokenManager(brokerType account.BrokerType, authorizationServerInfo AuthorizationServerInfo) TokenManager {
	factory.mutex.Lock()
	defer factory.mutex.Unlock()
	tokenManager, ok := factory.tokenManagerByBrokerType[brokerType]
	if !ok {
		tokenManager = NewTastyTradeTokenManager(&authorizationServerInfo)
		factory.tokenManagerByBrokerType[brokerType] = tokenManager
	}
	return tokenManager
}
