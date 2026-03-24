package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kduong/trading-backend/internal/account"
)

type TokenClaims struct {
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (manager *TokenManager) GenerateToken(object *account.Object) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(manager.ttl)
	claims := TokenClaims{
		AccountID: object.AccountID,
		Email:     object.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Subject:   object.AccountID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(manager.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}
