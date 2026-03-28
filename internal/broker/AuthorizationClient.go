package broker

import (
	"context"
)

type AuthorizationClient interface {
	BuildAuthorizationURL(stateToken string) string
	RequestAccessTokenUsingAuthorizationCode(ctx context.Context, code string) (*TokenOutput, error)
}

type TokenOutput struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}
