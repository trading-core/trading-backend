package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	"github.com/kduong/trading-backend/cmd/account-service/internal/broker"

	"github.com/kduong/trading-backend/cmd/account-service/internal/httpapi"

	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	var brokerCredentialsByType map[string]auth.Credentials
	data := config.EnvStringOrFatal("BROKER_CREDENTIALS_B64_JSON")
	reader := strings.NewReader(data)
	base64Decoder := base64.NewDecoder(base64.StdEncoding, reader)
	err := json.NewDecoder(base64Decoder).Decode(&brokerCredentialsByType)
	fatal.OnError(err)
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AccountStore: account.NewThreadSafeStoreDecorator(account.NewThreadSafeStoreDecoratorInput{
			Decorated: account.StoreFromEnv(ctx),
		}),
		BrokerAdapterFactory: broker.NewAdapterFactory(broker.NewAdapterFactoryInput{
			BrokerCredentialsByType: brokerCredentialsByType,
		}),
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
