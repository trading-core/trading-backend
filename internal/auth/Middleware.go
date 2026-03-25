package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/golang-jwt/jwt/v5"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/httputil"
)

type MiddleWare struct {
	TokenSecret string
}

func (middleware *MiddleWare) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		var err error
		defer func() {
			if err != nil {
				httputil.SendErrorResponse(responseWriter, err)
			}
		}()
		authHeader := strings.TrimSpace(request.Header.Get("Authorization"))
		if len(authHeader) == 0 {
			err = merry.New("missing authorization header").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			err = merry.New("invalid authorization header format").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		claims := &TokenClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(middleware.TokenSecret), nil
		})
		if err != nil || token == nil || !token.Valid {
			err = merry.New("invalid token").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		if len(claims.AccountID) == 0 {
			err = merry.New("account_id claim missing").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		ctx := contextx.WithAccountID(request.Context(), claims.AccountID)
		next.ServeHTTP(responseWriter, request.WithContext(ctx))
	})
}
