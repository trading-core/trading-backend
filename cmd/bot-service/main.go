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
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	logFactory, err := eventsource.LogFactoryFromEnv("BOT_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("bot:events")
	fatal.OnError(err)
	botChannelFunc := func(botID string) string {
		return fmt.Sprintf("bot:%s:events", botID)
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
	scalpingParams := tradingstrategy.ScalpingParams{
		MaxPositionFraction: config.EnvFloat64("BOT_SCALPING_MAX_POSITION_FRACTION", 0),
		TakeProfitPct:       config.EnvFloat64("BOT_SCALPING_TAKE_PROFIT_PCT", 0),
		SessionStart:        config.EnvInt("BOT_SCALPING_SESSION_START", 0),
		SessionEnd:          config.EnvInt("BOT_SCALPING_SESSION_END", 0),
		MinRSI:              config.EnvFloat64("BOT_SCALPING_MIN_RSI", 55),
		RequireMACDSignal:   config.EnvBool("BOT_SCALPING_REQUIRE_MACD_ABOVE_SIGNAL", true),
	}
	fatal.Unless(scalpingParams.MinRSI >= 0 && scalpingParams.MinRSI <= 100, "BOT_SCALPING_MIN_RSI must be in [0,100]")
	rsiPeriod := config.EnvInt("BOT_RSI_PERIOD", 14)
	macdFastPeriod := config.EnvInt("BOT_MACD_FAST_PERIOD", 12)
	macdSlowPeriod := config.EnvInt("BOT_MACD_SLOW_PERIOD", 26)
	macdSignalPeriod := config.EnvInt("BOT_MACD_SIGNAL_PERIOD", 9)
	fatal.Unless(rsiPeriod >= 2, "BOT_RSI_PERIOD must be at least 2")
	fatal.Unless(macdFastPeriod >= 2, "BOT_MACD_FAST_PERIOD must be at least 2")
	fatal.Unless(macdSlowPeriod > macdFastPeriod, "BOT_MACD_SLOW_PERIOD must be greater than BOT_MACD_FAST_PERIOD")
	fatal.Unless(macdSignalPeriod >= 2, "BOT_MACD_SIGNAL_PERIOD must be at least 2")
	botSyncActor := botsync.NewParentActor(botsync.NewParentActorInput{
		Log:                log,
		BotEventLogFactory: logFactory,
		BotChannelFunc:     botChannelFunc,
		ScalpingParams:     scalpingParams,
		RSIPeriod:          rsiPeriod,
		MACDFastPeriod:     macdFastPeriod,
		MACDSlowPeriod:     macdSlowPeriod,
		MACDSignalPeriod:   macdSignalPeriod,
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
