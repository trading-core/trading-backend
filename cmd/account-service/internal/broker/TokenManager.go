package broker

import (
	"context"
)

type TokenManager interface {
	GetAccessToken(ctx context.Context) (accessToken string, err error)
}

type GetAccessTokenOutput struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
}
