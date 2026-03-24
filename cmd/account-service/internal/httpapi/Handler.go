package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	"github.com/kduong/trading-backend/cmd/account-service/internal/broker"
)

type Handler struct {
	accountStore         account.Store
	brokerAdapterFactory *broker.AdapterFactory
}

type NewRouterInput struct {
	AccountStore         account.Store
	BrokerAdapterFactory *broker.AdapterFactory
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		accountStore:         input.AccountStore,
		brokerAdapterFactory: input.BrokerAdapterFactory,
	}
	router := mux.NewRouter().StrictSlash(true)
	accountV1Router := router.PathPrefix("/accounts/v1").Subrouter()
	// TODO: authorization
	accountV1Router.HandleFunc("/balance", handler.GetBalance).Methods(http.MethodGet).Name("GetBalance")
	return router
}
