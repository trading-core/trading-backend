package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"

	"github.com/kduong/trading-backend/cmd/account-service/internal/httpapi"

	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	var brokerCredentialsByType map[string]auth.Credentials
	data := config.EnvStringOrFatal("BROKER_CREDENTIALS_B64_JSON")
	reader := strings.NewReader(data)
	base64Decoder := base64.NewDecoder(base64.StdEncoding, reader)
	err := json.NewDecoder(base64Decoder).Decode(&brokerCredentialsByType)
	fatal.OnError(err)
	tastyTradeCredentials, ok := brokerCredentialsByType["tastytrade"]
	fatal.Unless(ok)
	tastyTradeAPIURL, err := url.Parse(tastyTradeCredentials.APIURL)
	fatal.OnError(err)
	tastyTradeTokenManager := auth.NewTastyTradeTokenManager(&tastyTradeCredentials.AuthorizationServer)
	logFactory, err := eventsource.LogFactoryFromEnv("LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("account:events")
	fatal.OnError(err)
	authorizationRedirectURI := url.URL{
		Scheme: config.EnvStringOrFatal("TRADING_API_SCHEME"),
		Host:   config.EnvStringOrFatal("TRADING_API_HOST"),
		Path:   "/accounts/v1/authorization_callback",
	}
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AccountStore: account.NewThreadSafeStoreDecorator(account.NewThreadSafeStoreDecoratorInput{
			Decorated: account.NewEventSourcedStore(account.NewEventSourcedStoreInput{
				Log: log,
			}),
		}),
		BrokerClientFactory: &broker.ClientFactory{
			TastyTradeClientFactory: &tastytrade.HTTPClientFactory{
				APIURL:         tastyTradeAPIURL,
				GetAccessToken: tastyTradeTokenManager.GetAccessToken,
			},
		},
		AuthMiddleWare: &auth.MiddleWare{
			TokenSecret: config.EnvStringOrFatal("TOKEN_SECRET"),
		},
		BackendRedirectURI:    authorizationRedirectURI.String(),
		TastyTradeCredentials: tastyTradeCredentials,
		FrontendBaseURL:       config.EnvStringOrFatal("FRONTEND_BASE_URL"),
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
