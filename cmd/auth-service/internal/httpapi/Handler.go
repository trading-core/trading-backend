package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/auth-service/internal/auth"
	"github.com/kduong/trading-backend/internal/account"
)

type Handler struct {
	accountStore account.Store
	tokenManager *auth.TokenManager
}

type NewRouterInput struct {
	AccountStore account.Store
	TokenManager *auth.TokenManager
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		accountStore: input.AccountStore,
		tokenManager: input.TokenManager,
	}
	router := mux.NewRouter().StrictSlash(true)
	authV1Router := router.PathPrefix("/auth/v1").Subrouter()
	authV1Router.HandleFunc("/accounts", handler.CreateAccount).Methods(http.MethodPost).Name("CreateAccount")
	authV1Router.HandleFunc("/login", handler.Login).Methods(http.MethodPost).Name("Login")
	authV1Router.HandleFunc("/accounts", handler.ListAccounts).Methods(http.MethodGet).Name("ListAccounts")
	authV1Router.HandleFunc("/accounts/{account_id}", handler.GetAccount).Methods(http.MethodGet).Name("GetAccount")
	return router
}
