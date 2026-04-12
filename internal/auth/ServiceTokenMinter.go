package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kduong/trading-backend/internal/config"
)

const serviceUserID = "service"

// ServiceTokenMinter mints short-lived JWTs for internal service-to-service calls.
type ServiceTokenMinter struct {
	tokenSecret []byte
	ttl         time.Duration
}

type NewServiceTokenMinterInput struct {
	TokenSecret []byte
	TTL         time.Duration
}

func NewServiceTokenMinter(input NewServiceTokenMinterInput) *ServiceTokenMinter {
	return &ServiceTokenMinter{
		tokenSecret: input.TokenSecret,
		ttl:         input.TTL,
	}
}

func ServiceTokenMinterFromEnv() *ServiceTokenMinter {
	return NewServiceTokenMinter(NewServiceTokenMinterInput{
		TokenSecret: []byte(config.EnvStringOrFatal("TOKEN_SECRET")),
		TTL:         config.EnvDuration("SERVICE_TOKEN_TTL", 5*time.Minute),
	})
}

// MintToken returns a signed JWT with subject "service" valid for the configured TTL.
func (minter *ServiceTokenMinter) MintToken() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   serviceUserID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(minter.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(minter.tokenSecret)
}
