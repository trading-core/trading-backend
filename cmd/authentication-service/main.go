package main

import (
	"context"
	"net/http"
	"time"

	"github.com/kduong/trading-backend/cmd/authentication-service/internal/httpapi"
	"github.com/kduong/trading-backend/cmd/authentication-service/internal/user"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		UserStore: user.NewThreadSafeStoreDecorator(user.NewThreadSafeStoreDecoratorInput{
			Decorated: user.StoreFromEnv(ctx),
		}),
		TokenSecret: []byte(config.EnvStringOrFatal("TOKEN_SECRET")),
		ExpiryTTL:   1 * time.Hour,
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
