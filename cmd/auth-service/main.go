package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/kduong/trading-backend/cmd/auth-service/internal/auth"
	"github.com/kduong/trading-backend/cmd/auth-service/internal/httpapi"
	"github.com/kduong/trading-backend/internal/account"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	var accountStore account.Store
	storeDriver := strings.ToLower(config.EnvString("AUTH_STORE_DRIVER", "memory"))
	if storeDriver == "postgres" {
		postgresStore, err := account.NewPostgresStore(ctx, config.EnvStringOrFatal("AUTH_POSTGRES_DSN"))
		fatal.OnError(err)
		accountStore = postgresStore
	} else {
		accountStore = account.NewThreadSafeStoreDecorator(account.NewThreadSafeStoreDecoratorInput{
			Decorated: account.NewInMemoryStore(),
		})
	}
	tokenManager := auth.NewTokenManager(
		config.EnvString("AUTH_JWT_SECRET", "local-dev-auth-secret"),
		time.Duration(config.EnvInt("AUTH_JWT_TTL_MINUTES", 60))*time.Minute,
	)
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AccountStore: accountStore,
		TokenManager: tokenManager,
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Range", "If-Range"},
		ExposedHeaders:   []string{"Set-Cookie", "Allow", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified"},
		AllowCredentials: true,
	})
	http.ListenAndServe(":9100", c.Handler(router))
}
