package broker

import (
	"context"
)

type AuthorizationClient interface {
	BuildAuthorizationURL(stateToken string) (string, error)
	ExchangeCode(ctx context.Context, code string) (*AuthorizationTokens, error)
	ListAccounts(ctx context.Context, accessToken string) ([]string, error)
	GenerateAccount(accountID string) (*Account, error)
}

type AuthorizationTokens struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}
