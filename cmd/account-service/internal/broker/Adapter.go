package broker

import (
	"context"
	"sync"

	"github.com/kduong/trading-backend/internal/account"
)

type Adapter interface {
	GetBalanceInfo(ctx context.Context) (*BalanceInfo, error)
}

type BalanceInfo struct {
	AccountBroker account.BrokerType `json:"account_broker"`
	Balance       float64            `json:"balance"`
	Currency      string             `json:"currency"`
}

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
			Account:        object,
			RawAPIURL:      credentials.APIURL,
			GetAccessToken: tokenManager.GetAccessToken,
		})
	default:
		panic("Unsupported broker type: " + object.BrokerType)
	}
}

var tokenManagerFactory = map[account.BrokerType]func(authorizationServerInfo AuthorizationServerInfo) TokenManager{
	account.BrokerTypeTastyTrade: func(authorizationServerInfo AuthorizationServerInfo) TokenManager {
		return NewTastyTradeTokenManager(&authorizationServerInfo)
	},
}

func (factory *AdapterFactory) getOrCreateTokenManager(brokerType account.BrokerType, authorizationServerInfo AuthorizationServerInfo) TokenManager {
	factory.mutex.Lock()
	defer factory.mutex.Unlock()
	tokenManager, ok := factory.tokenManagerByBrokerType[brokerType]
	if !ok {
		tokenManager = tokenManagerFactory[brokerType](authorizationServerInfo)
		factory.tokenManagerByBrokerType[brokerType] = tokenManager
	}
	return tokenManager
}
