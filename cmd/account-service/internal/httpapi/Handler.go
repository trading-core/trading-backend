package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/account-service/internal/broker"
	"github.com/kduong/trading-backend/internal/account"
)

type Handler struct {
	accountStore           account.Store
	brokerAdapterFactory   *broker.AdapterFactory
	authJWTSecret          string
	defaultBrokerType      account.BrokerType
	defaultBrokerAccountID string
}

type NewRouterInput struct {
	AccountStore           account.Store
	BrokerAdapterFactory   *broker.AdapterFactory
	AuthJWTSecret          string
	DefaultBrokerType      account.BrokerType
	DefaultBrokerAccountID string
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		accountStore:           input.AccountStore,
		brokerAdapterFactory:   input.BrokerAdapterFactory,
		authJWTSecret:          input.AuthJWTSecret,
		defaultBrokerType:      input.DefaultBrokerType,
		defaultBrokerAccountID: input.DefaultBrokerAccountID,
	}
	router := mux.NewRouter().StrictSlash(true)
	accountV1Router := router.PathPrefix("/accounts/v1").Subrouter()
	// TODO: authorization
	accountV1Router.HandleFunc("/balance", handler.GetBalance).Methods(http.MethodGet).Name("GetBalance")
	return router
}
