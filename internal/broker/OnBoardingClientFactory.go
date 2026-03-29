package broker

import "context"

type OnBoardingClientFactory interface {
	GetAuthorizationClient(accountType AccountType) (authorizationClient AuthorizationClient, err error)
	GetAccountDiscoveryClient(ctx context.Context, accountType AccountType) (accountDiscoveryClient AccountDiscoveryClient, err error)
}
