package main

import (
	"context"
	"net/http"
	"net/url"

	"github.com/kduong/trading-backend/cmd/account-service/pkg/accountservice"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botsync"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/httpapi"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	logFactory, err := eventsource.LogFactoryFromEnv("BOT_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("bot:events")
	fatal.OnError(err)
	credentialsByType := auth.CredentialsByTypeFromEnv()
	tastyTradeAPIURL, tastyTradeTokenManager := loadTastyTradeConfiguration(credentialsByType, "tastytrade")
	tastyTradeSandboxAPIURL, tastyTradeSandboxTokenManager := loadTastyTradeConfiguration(credentialsByType, "tastytrade_sandbox")
	botSyncActor := botsync.NewParentActor(botsync.NewParentActorInput{
		Log:                log,
		BotEventLogFactory: logFactory,
		BrokerAccountClientFactory: &BrokerAccountClientFactory{
			TastyTradeClientFactory: &tastytrade.HTTPClientFactory{
				APIURL:         tastyTradeAPIURL,
				GetAccessToken: tastyTradeTokenManager.GetAccessToken,
			},
			TastyTradeSandboxClientFactory: &tastytrade.HTTPClientFactory{
				APIURL:         tastyTradeSandboxAPIURL,
				GetAccessToken: tastyTradeSandboxTokenManager.GetAccessToken,
			},
		},
		BrokerMarketDataClientFactory: &BrokerMarketDataClientFactory{
			TastyTradeClientFactory: &tastytrade.HTTPClientFactory{
				APIURL:         tastyTradeAPIURL,
				GetAccessToken: tastyTradeTokenManager.GetAccessToken,
			},
			TastyTradeSandboxClientFactory: &tastytrade.HTTPClientFactory{
				APIURL:         tastyTradeSandboxAPIURL,
				GetAccessToken: tastyTradeSandboxTokenManager.GetAccessToken,
			},
		},
	})
	go func() {
		cursor := botSyncActor.CatchUp(ctx)
		_, err = subscription.Live(ctx, subscription.Input{
			Log:    log,
			Cursor: cursor,
			Apply:  botSyncActor.Apply,
		})
		fatal.OnError(err)
	}()
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AuthMiddleware:       auth.MiddlewareFromEnv(),
		AccountServiceClient: accountservice.ClientFromEnv(),
		BotEventLogFactory:   logFactory,
		BotStoreCommandHandler: botstore.NewCommandHandlerThreadSafeDecorator(botstore.NewCommandHandlerThreadSafeDecoratorInput{
			Decorated: botstore.NewEventSourcedCommandHandler(botstore.NewEventSourcedCommandHandlerInput{
				Log: log,
			}),
		}),
		BotStoreQueryHandler: botstore.NewQueryHandlerThreadSafeDecorator(botstore.NewQueryHandlerThreadSafeDecoratorInput{
			Decorated: botstore.NewEventSourcedQueryHandler(botstore.NewEventSourcedQueryHandlerInput{
				Log: log,
			}),
		}),
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Range", "If-Range"},
		ExposedHeaders:   []string{"Set-Cookie", "Allow", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified"},
		AllowCredentials: true,
	})
	http.ListenAndServe(":8080", c.Handler(router))
}

func loadTastyTradeConfiguration(credentialsByType map[string]auth.Credentials, brokerType string) (*url.URL, *auth.TastyTradeTokenManager) {
	credentials, ok := credentialsByType[brokerType]
	fatal.Unless(ok)
	apiURL, err := url.Parse(credentials.APIURL)
	fatal.OnError(err)
	tokenManager := auth.NewTastyTradeTokenManager(&credentials.AuthorizationServer)
	return apiURL, tokenManager
}
