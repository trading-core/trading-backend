package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/golang-jwt/jwt/v5"

	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/httpx"
)

type Middleware struct {
	tokenSecret []byte
	// audience, if non-empty, is the service name this middleware is protecting.
	// A presented token must either carry no audience (direct user token) or
	// include this value. Leave empty to accept any audience.
	audience string
}

type NewMiddlewareInput struct {
	TokenSecret []byte
	Audience    string
}

func NewMiddleware(input NewMiddlewareInput) *Middleware {
	return &Middleware{
		tokenSecret: input.TokenSecret,
		audience:    input.Audience,
	}
}

// MiddlewareFromEnv builds a middleware whose audience is the given service
// identifier. Pass an empty string to disable audience enforcement.
func MiddlewareFromEnv(audience string) *Middleware {
	return NewMiddleware(NewMiddlewareInput{
		TokenSecret: []byte(config.EnvStringOrFatal("TOKEN_SECRET")),
		Audience:    audience,
	})
}

func (middleware *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		var err error
		defer func() {
			if err != nil {
				httpx.SendErrorResponse(responseWriter, err)
			}
		}()
		authorization := request.Header.Get("Authorization")
		if len(authorization) == 0 {
			err = merry.New("missing authorization header").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			err = merry.New("invalid authorization header format").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		claims := new(Claims)
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return middleware.tokenSecret, nil
		})
		if err != nil || token == nil || !token.Valid {
			err = merry.New("invalid token").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		if len(claims.Subject) == 0 {
			err = merry.New("subject claim missing").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		if !middleware.audienceAllowed(claims.Audience) {
			err = merry.New("token audience does not include this service").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		ctx := request.Context()
		ctx = contextx.WithUserID(ctx, claims.Subject)
		ctx = contextx.WithScopes(ctx, claims.Scopes())
		if claims.Act != nil {
			ctx = contextx.WithActor(ctx, claims.Act.Sub)
		}
		next.ServeHTTP(responseWriter, request.WithContext(ctx))
	})
}

// audienceAllowed returns true when this middleware should accept a token with
// the given audience list. Tokens without an audience (direct user tokens) are
// always accepted; tokens with an audience must include this service.
func (middleware *Middleware) audienceAllowed(tokenAudience jwt.ClaimStrings) bool {
	if middleware.audience == "" {
		return true
	}
	if len(tokenAudience) == 0 {
		return true
	}
	for _, audience := range tokenAudience {
		if audience == middleware.audience {
			return true
		}
	}
	return false
}
