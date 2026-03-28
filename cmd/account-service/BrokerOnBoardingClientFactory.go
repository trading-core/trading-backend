package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
	"github.com/kduong/trading-backend/internal/fatal"
)

type BrokerOnboardingClientFactory struct {
	BackendRedirectURI string
	CredentialsByType  map[broker.AccountType]auth.Credentials
}

func (factory *BrokerOnboardingClientFactory) GetAuthorizationClient(accountType broker.AccountType) (authorizationClient broker.AuthorizationClient, err error) {
	switch accountType {
	case broker.AccountTypeTastyTrade:
		authorizationClient = broker.NewTastyTradeAuthorizationClient(broker.TastyTradeAuthorizationClientInput{
			BackendRedirectURI: factory.BackendRedirectURI,
			Credentials:        factory.CredentialsByType[accountType],
		})
		return
	default:
		err = fmt.Errorf("unsupported broker type: %s", accountType)
		return
	}
}

func (factory *BrokerOnboardingClientFactory) GetAccountDiscoveryClient(accountType broker.AccountType, accessToken string) (accountDiscoveryClient broker.AccountDiscoveryClient, err error) {
	switch accountType {
	case broker.AccountTypeTastyTrade:
		var apiURL *url.URL
		apiURL, err = url.Parse(factory.CredentialsByType[accountType].APIURL)
		fatal.OnError(err)
		accountDiscoveryClient = broker.NewTastyTradeAccountDiscoveryAdapter(broker.TastyTradeAccountDiscoveryAdapterInput{
			Client: tastytrade.NewHTTPClient(tastytrade.NewHTTPClientInput{
				APIURL: apiURL,
				GetAccessToken: func(ctx context.Context) (string, error) {
					return accessToken, nil
				},
			}),
		})
		return
	default:
		err = fmt.Errorf("unsupported broker type: %s", accountType)
		return
	}
}
