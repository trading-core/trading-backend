package auth

import "github.com/golang-jwt/jwt/v5"

type TokenClaims struct {
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}
