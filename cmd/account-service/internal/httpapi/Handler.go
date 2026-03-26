package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
)

type Handler struct {
	accountStore        account.Store
	brokerClientFactory *broker.ClientFactory
}

type NewRouterInput struct {
	AccountStore        account.Store
	BrokerClientFactory *broker.ClientFactory
	AuthMiddleWare      *auth.MiddleWare
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		accountStore:        input.AccountStore,
		brokerClientFactory: input.BrokerClientFactory,
	}
	router := mux.NewRouter().StrictSlash(true)
	accountV1Router := router.PathPrefix("/accounts/v1").Subrouter()
	accountV1Router.Use(input.AuthMiddleWare.Handle)
	accountV1Router.HandleFunc("/accounts", handler.CreateAccount).Methods(http.MethodPost).Name("CreateAccount")
	accountV1Router.HandleFunc("/accounts", handler.ListAccounts).Methods(http.MethodGet).Name("ListAccounts")
	accountV1Router.HandleFunc("/accounts/{account_id}", handler.GetAccount).Methods(http.MethodGet).Name("GetAccount")
	accountV1Router.HandleFunc("/accounts/{account_id}/balances", handler.GetAccountBalance).Methods(http.MethodGet).Name("GetAccountBalance")
	accountV1Router.HandleFunc("/accounts/{account_id}/broker", handler.LinkBroker).Methods(http.MethodPost).Name("LinkBroker")
	return router
}
