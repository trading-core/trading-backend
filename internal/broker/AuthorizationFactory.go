package broker

import (
	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/internal/auth"
)

type AuthorizationClientFactory struct {
	BackendRedirectURI string
	CredentialsByType  map[AccountType]auth.Credentials
}

func (factory *AuthorizationClientFactory) Get(accountType AccountType) (AuthorizationClient, error) {
	credentials, ok := factory.CredentialsByType[accountType]
	if !ok {
		return nil, merry.New("unsupported broker type: " + string(accountType))
	}
	switch accountType {
	case AccountTypeTastyTrade:
		return NewTastyTradeAuthorizationClient(TastyTradeAuthorizationClientInput{
			BackendRedirectURI: factory.BackendRedirectURI,
			Credentials:        credentials,
		}), nil
	default:
		return nil, merry.New("unsupported broker type: " + string(accountType))
	}
}
