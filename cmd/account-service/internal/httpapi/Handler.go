package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
)

type Handler struct {
	accountStore          account.Store
	brokerClientFactory   *broker.ClientFactory
	backendRedirectURI    string
	tastyTradeCredentials auth.Credentials
	frontendBaseURL       string
}

type NewRouterInput struct {
	AccountStore          account.Store
	BrokerClientFactory   *broker.ClientFactory
	AuthMiddleWare        *auth.MiddleWare
	BackendRedirectURI    string
	TastyTradeCredentials auth.Credentials
	FrontendBaseURL       string
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		accountStore:          input.AccountStore,
		brokerClientFactory:   input.BrokerClientFactory,
		backendRedirectURI:    input.BackendRedirectURI,
		tastyTradeCredentials: input.TastyTradeCredentials,
		frontendBaseURL:       input.FrontendBaseURL,
	}
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/accounts/v1/authorization_callback", handler.HandleAuthorizationCallback).Methods(http.MethodGet).Name("HandleAuthorizationCallback")

	accountV1Router := router.PathPrefix("/accounts/v1").Subrouter()
	accountV1Router.Use(input.AuthMiddleWare.Handle)
	accountV1Router.HandleFunc("/accounts", handler.CreateAccount).Methods(http.MethodPost).Name("CreateAccount")
	accountV1Router.HandleFunc("/accounts", handler.ListAccounts).Methods(http.MethodGet).Name("ListAccounts")
	accountV1Router.HandleFunc("/accounts/{account_id}", handler.GetAccount).Methods(http.MethodGet).Name("GetAccount")
	accountV1Router.HandleFunc("/accounts/{account_id}/balances", handler.GetAccountBalance).Methods(http.MethodGet).Name("GetAccountBalance")

	accountV1Router.HandleFunc("/accounts/{account_id}/brokers", handler.StartBrokerSelection).Methods(http.MethodPost).Name("StartBrokerSelection")
	accountV1Router.HandleFunc("/accounts/{account_id}/brokers", handler.GetPendingBrokerSelection).Methods(http.MethodGet).Name("GetPendingBrokerSelection")
	accountV1Router.HandleFunc("/accounts/{account_id}/brokers", handler.CompleteBrokerSelection).Methods(http.MethodPut).Name("CompleteBrokerSelection")
	return router
}
