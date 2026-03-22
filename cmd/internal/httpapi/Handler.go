package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/internal/account"
	"github.com/kduong/trading-backend/cmd/internal/broker"
)

type Handler struct {
	accountObjectStore   account.ObjectStore
	brokerAdapterFactory *broker.AdapterFactory
}

type NewRouterInput struct {
	AccountObjectStore   account.ObjectStore
	BrokerAdapterFactory *broker.AdapterFactory
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		accountObjectStore:   input.AccountObjectStore,
		brokerAdapterFactory: input.BrokerAdapterFactory,
	}
	router := mux.NewRouter().StrictSlash(true)
	accountV1Router := router.PathPrefix("/account/v1").Subrouter()
	// TODO: authorization
	accountV1Router.HandleFunc("/balance", handler.GetBalance).Methods(http.MethodGet).Name("GetBalance")
	return router
}
