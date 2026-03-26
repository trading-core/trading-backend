package auth

import (
	"context"
)

type TokenManager interface {
	GetAccessToken(ctx context.Context) (accessToken string, err error)
}
