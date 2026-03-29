package main

import (
	"net/http"

	"github.com/kduong/trading-backend/cmd/account-service/pkg/accountservice"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/httpapi"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	logFactory, err := eventsource.LogFactoryFromEnv("BOT_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("bot:events")
	fatal.OnError(err)
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AuthMiddleware:       auth.MiddlewareFromEnv(),
		AccountServiceClient: accountservice.ClientFromEnv(),
		BotStore: botstore.NewThreadSafeDecorator(botstore.NewThreadSafeDecoratorInput{
			Decorated: botstore.NewEventSourcedStore(botstore.NewEventSourcedStoreInput{
				Log: log,
			}),
		}),
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Range", "If-Range"},
		ExposedHeaders:   []string{"Set-Cookie", "Allow", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified"},
		AllowCredentials: true,
	})
	http.ListenAndServe(":8080", c.Handler(router))
}
