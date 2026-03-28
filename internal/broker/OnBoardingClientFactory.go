package broker

type OnBoardingClientFactory interface {
	GetAuthorizationClient(accountType AccountType) (authorizationClient AuthorizationClient, err error)
	GetAccountDiscoveryClient(accountType AccountType, accessToken string) (accountDiscoveryClient AccountDiscoveryClient, err error)
}
