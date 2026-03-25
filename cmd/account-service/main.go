package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"

	"github.com/kduong/trading-backend/cmd/account-service/internal/httpapi"

	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AccountStore: account.NewThreadSafeStoreDecorator(account.NewThreadSafeStoreDecoratorInput{
			Decorated: account.StoreFromEnv(ctx),
		}),
		BrokerAdapterFactory: &broker.AdapterFactory{
			BrokerClientByType: BrokerClientByTypeFromEnv(),
		},
		AuthMiddleWare: &auth.MiddleWare{
			TokenSecret: config.EnvStringOrFatal("TOKEN_SECRET"),
		},
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

func BrokerClientByTypeFromEnv() map[string]broker.Client {
	var brokerCredentialsByType map[string]auth.Credentials
	data := config.EnvStringOrFatal("BROKER_CREDENTIALS_B64_JSON")
	reader := strings.NewReader(data)
	base64Decoder := base64.NewDecoder(base64.StdEncoding, reader)
	err := json.NewDecoder(base64Decoder).Decode(&brokerCredentialsByType)
	fatal.OnError(err)
	brokerClientByType := make(map[string]broker.Client)
	for brokerType, credentials := range brokerCredentialsByType {
		apiURL, err := url.Parse(credentials.APIURL)
		fatal.OnError(err)
		brokerClientByType[brokerType] = broker.Client{
			APIURL:       apiURL,
			TokenManager: auth.TokenManagerFactory[brokerType](&credentials.AuthorizationServer),
		}
	}
	return brokerClientByType
}
