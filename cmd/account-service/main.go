package main

import (
	"net/http"
	"net/url"

	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"

	"github.com/kduong/trading-backend/cmd/account-service/internal/httpapi"
	"github.com/kduong/trading-backend/cmd/account-service/internal/oauthstatestore"
	"github.com/kduong/trading-backend/cmd/account-service/internal/pendingselectionstore"

	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	logFactory, err := eventsource.LogFactoryFromEnv("ACCOUNT_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("account:events")
	fatal.OnError(err)
	authorizationRedirectURI := url.URL{
		Scheme: config.EnvStringOrFatal("TRADING_API_SCHEME"),
		Host:   config.EnvStringOrFatal("TRADING_API_HOST"),
		Path:   "/accounts/v1/authorization_callback",
	}
	credentialsByType := auth.CredentialsByTypeFromEnv()
	tastyTradeCredentials, tastyTradeAPIURL, tastyTradeTokenManager := LoadTastyTradeConfiguration(credentialsByType, "tastytrade")
	tastyTradeSandboxCredentials, tastyTradeSandboxAPIURL, tastyTradeSandboxTokenManager := LoadTastyTradeConfiguration(credentialsByType, "tastytrade_sandbox")
	brokerAuthorizationCredentials := map[broker.AccountType]auth.Credentials{
		broker.AccountTypeTastyTrade:        tastyTradeCredentials,
		broker.AccountTypeTastyTradeSandbox: tastyTradeSandboxCredentials,
	}
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		OAuthStateStore:       oauthstatestore.NewInMemory(),
		PendingSelectionStore: pendingselectionstore.NewInMemory(),
		AccountStoreCommandHandler: accountstore.NewCommandHandlerThreadSafeDecorator(accountstore.NewCommandHandlerThreadSafeDecoratorInput{
			Decorated: accountstore.NewEventSourcedCommandHandler(accountstore.NewEventSourcedCommandHandlerInput{
				Log: log,
			}),
		}),
		AccountStoreQueryHandler: accountstore.NewQueryHandlerThreadSafeDecorator(accountstore.NewQueryHandlerThreadSafeDecoratorInput{
			Decorated: accountstore.NewEventSourcedQueryHandler(accountstore.NewEventSourcedQueryHandlerInput{
				Log: log,
			}),
		}),
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
		BrokerOnBoardingClientFactory: &BrokerOnboardingClientFactory{
			BackendRedirectURI: authorizationRedirectURI.String(),
			CredentialsByType:  brokerAuthorizationCredentials,
		},
		AuthMiddleware:     auth.MiddlewareFromEnv(),
		BackendRedirectURI: authorizationRedirectURI.String(),
		FrontendBaseURL:    config.EnvStringOrFatal("FRONTEND_BASE_URL"),
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Range", "If-Range"},
		ExposedHeaders:   []string{"Set-Cookie", "Allow", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified"},
		AllowCredentials: true,
	})
	http.ListenAndServe(":9000", c.Handler(router))
}

func LoadTastyTradeConfiguration(credentialsByType map[string]auth.Credentials, brokerType string) (auth.Credentials, *url.URL, *auth.TastyTradeTokenManager) {
	credentials, ok := credentialsByType[brokerType]
	fatal.Unless(ok)
	apiURL, err := url.Parse(credentials.APIURL)
	fatal.OnError(err)
	tokenManager := auth.NewTastyTradeTokenManager(&credentials.AuthorizationServer)
	return credentials, apiURL, tokenManager
}
