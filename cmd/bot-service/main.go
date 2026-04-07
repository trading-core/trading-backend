package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kduong/trading-backend/cmd/account-service/pkg/accountservice"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botsync"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/brokerfactory"
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
	logFactory, err := eventsource.LogFactoryFromEnv("TRADING_BOT_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("trading_bot:events")
	fatal.OnError(err)
	botChannelFunc := func(botID string) string {
		return fmt.Sprintf("trading_bot:%s:events", botID)
	}
	credentialsByType := auth.CredentialsByTypeFromEnv()
	tastyTradeAPIURL, tastyTradeTokenManager := loadTastyTradeConfiguration(credentialsByType, "tastytrade")
	tastyTradeSandboxAPIURL, tastyTradeSandboxTokenManager := loadTastyTradeConfiguration(credentialsByType, "tastytrade_sandbox")
	tastyTradeClientFactory := &tastytrade.HTTPClientFactory{
		APIURL:         tastyTradeAPIURL,
		GetAccessToken: tastyTradeTokenManager.GetAccessToken,
	}
	tastyTradeSandboxClientFactory := &tastytrade.HTTPClientFactory{
		APIURL:         tastyTradeSandboxAPIURL,
		GetAccessToken: tastyTradeSandboxTokenManager.GetAccessToken,
	}
	symbolValidator := brokerfactory.NewSymbolValidator(brokerfactory.NewSymbolValidatorInput{
		TastyTradeClientFactory:        tastyTradeClientFactory,
		TastyTradeSandboxClientFactory: tastyTradeSandboxClientFactory,
	})
	botSyncActor := botsync.NewParentActor(botsync.NewParentActorInput{
		Log:                    log,
		BotEventLogFactory:     logFactory,
		BotChannelFunc:         botChannelFunc,
		RSIPeriod:              14,   // RSI period at 14 days
		MACDFastPeriod:         12,   // Default MACD fast period.
		MACDSlowPeriod:         26,   // Default MACD slow period.
		MACDSignalPeriod:       9,    // Default MACD signal period.
		BollingerPeriod: 20,  // Common Bollinger Bands period.
		BollingerStdDev: 2.0, // Common Bollinger Bands standard deviation multiplier.
		BrokerAccountClientFactory: &brokerfactory.AccountClientFactory{
			TastyTradeClientFactory:        tastyTradeClientFactory,
			TastyTradeSandboxClientFactory: tastyTradeSandboxClientFactory,
		},
		BrokerMarketDataClientFactory: &brokerfactory.MarketDataClientFactory{
			TastyTradeClientFactory:        tastyTradeClientFactory,
			TastyTradeSandboxClientFactory: tastyTradeSandboxClientFactory,
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
		SymbolValidator:      symbolValidator,
		BotEventLogFactory:   logFactory,
		BotChannelFunc:       botChannelFunc,
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
	err = http.ListenAndServe(":8081", c.Handler(router))
	fatal.OnError(err)
}

func loadTastyTradeConfiguration(credentialsByType map[string]auth.Credentials, brokerType string) (*url.URL, *auth.TastyTradeTokenManager) {
	credentials, ok := credentialsByType[brokerType]
	fatal.Unless(ok)
	apiURL, err := url.Parse(credentials.APIURL)
	fatal.OnError(err)
	tokenManager := auth.NewTastyTradeTokenManager(&credentials.AuthorizationServer)
	return apiURL, tokenManager
}
