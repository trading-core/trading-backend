package auth

import (
	"context"
)

type TokenManager interface {
	GetAccessToken(ctx context.Context) (accessToken string, err error)
}

var TokenManagerFactory = map[string]func(config *AuthorizationServerInfo) TokenManager{
	"tastytrade": func(config *AuthorizationServerInfo) TokenManager {
		return NewTastyTradeTokenManager(config)
	},
}
