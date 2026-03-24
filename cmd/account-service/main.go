package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kduong/trading-backend/cmd/internal/account"
	"github.com/kduong/trading-backend/cmd/internal/broker"
	"github.com/kduong/trading-backend/cmd/internal/httpapi"
	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	accountStore := account.NewThreadSafeStoreDecorator(account.NewThreadSafeStoreDecoratorInput{
		Decorated: account.NewInMemoryStore(),
	})
	// Test data
	accountStore.Put(ctx, &account.Object{
		AccountID:  "TEST",
		BrokerType: account.BrokerTypeMockTest,
	})
	// Tasty Trade account
	accountStore.Put(ctx, &account.Object{
		AccountID:       "TastyTradeAccountID",
		BrokerType:      account.BrokerTypeTastyTrade,
		BrokerAccountID: "6AB16514",
	})
	var brokerCredentialsByType map[account.BrokerType]broker.Credentials
	data := config.EnvStringOrFatal("BROKER_CREDENTIALS_B64_JSON")
	reader := strings.NewReader(data)
	base64Decoder := base64.NewDecoder(base64.StdEncoding, reader)
	err := json.NewDecoder(base64Decoder).Decode(&brokerCredentialsByType)
	fatal.OnError(err)
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AccountStore: accountStore,
		BrokerAdapterFactory: broker.NewAdapterFactory(broker.NewAdapterFactoryInput{
			BrokerCredentialsByType: brokerCredentialsByType,
		}),
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
