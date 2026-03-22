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
	accountObjectStore := account.NewLockingDecorator(account.NewLockingDecoratorInput{
		Decorated: NewTestAccountObjectStore(NewTestAccountObjectStoreInput{
			Decorated: account.NewInMemoryObjectStore(),
		}),
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

var _ account.ObjectStore = (*TestAccountObjectStore)(nil)

type TestAccountObjectStore struct {
	decorated account.ObjectStore
}

type NewTestAccountObjectStoreInput struct {
	Decorated account.ObjectStore
}

func NewTestAccountObjectStore(input NewTestAccountObjectStoreInput) *TestAccountObjectStore {
	return &TestAccountObjectStore{
		decorated: input.Decorated,
	}
}

func (decorator *TestAccountObjectStore) GetObject(ctx context.Context, accountID string) (object *account.Object, err error) {
	if accountID == "TEST" {
		object = &account.Object{
			ID:         "TEST",
			BrokerType: account.BrokerTypeMockTest,
		}
		return
	}
	return decorator.decorated.GetObject(ctx, accountID)
}
