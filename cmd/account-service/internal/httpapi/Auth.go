package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/golang-jwt/jwt/v5"
)

type tokenClaims struct {
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}

func (handler *Handler) extractAccountID(request *http.Request) (string, error) {
	authHeader := strings.TrimSpace(request.Header.Get("Authorization"))
	if len(authHeader) == 0 {
		return "", merry.New("missing authorization header").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("authorization required")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", merry.New("invalid authorization header format").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("invalid authorization header")
	}
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(handler.authJWTSecret), nil
	})
	if err != nil || token == nil || !token.Valid {
		return "", merry.New("invalid token").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("invalid token")
	}
	if len(claims.AccountID) == 0 {
		return "", merry.New("account_id claim missing").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("invalid token claims")
	}
	return claims.AccountID, nil
}
