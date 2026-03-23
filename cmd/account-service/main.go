package main

import (
	"context"
	"net/http"

	"github.com/kduong/trading-backend/cmd/internal/account"
	"github.com/kduong/trading-backend/cmd/internal/broker"
	"github.com/kduong/trading-backend/cmd/internal/httpapi"
	"github.com/rs/cors"
)

func main() {
	ctx := context.Background()
	accountObjectStore := account.NewThreadSafeObjectStoreDecorator(account.NewThreadSafeObjectStoreDecoratorInput{
		Decorated: account.NewInMemoryObjectStore(),
	})
	// Test data
	accountObjectStore.Put(ctx, &account.Object{
		AccountID:  "TEST",
		BrokerType: account.BrokerTypeMockTest,
	})
	// Tasty Trade account
	accountObjectStore.Put(ctx, &account.Object{
		AccountID:  "TastyTradeAccountID",
		BrokerType: account.BrokerTypeTastyTrade,
	})
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AccountObjectStore:   accountObjectStore,
		BrokerAdapterFactory: new(broker.AdapterFactory),
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
