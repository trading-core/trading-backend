package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/cmd/account-service/internal/oauthstatestore"
	"github.com/kduong/trading-backend/cmd/account-service/internal/pendingselectionstore"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
)

type Handler struct {
	oauthStateStore               oauthstatestore.Store
	pendingSelectionStore         pendingselectionstore.Store
	accountStoreCommandHandler    accountstore.CommandHandler
	accountStoreQueryHandler      accountstore.QueryHandler
	brokerAccountClientFactory    broker.AccountClientFactory
	brokerOnBoardingClientFactory broker.OnBoardingClientFactory
	backendRedirectURI            string
	frontendBaseURL               string
}

type NewRouterInput struct {
	OAuthStateStore               oauthstatestore.Store
	PendingSelectionStore         pendingselectionstore.Store
	AccountStoreCommandHandler    accountstore.CommandHandler
	AccountStoreQueryHandler      accountstore.QueryHandler
	BrokerAccountClientFactory    broker.AccountClientFactory
	BrokerOnBoardingClientFactory broker.OnBoardingClientFactory
	AuthMiddleware                *auth.Middleware
	BackendRedirectURI            string
	FrontendBaseURL               string
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		oauthStateStore:               input.OAuthStateStore,
		pendingSelectionStore:         input.PendingSelectionStore,
		accountStoreCommandHandler:    input.AccountStoreCommandHandler,
		accountStoreQueryHandler:      input.AccountStoreQueryHandler,
		brokerAccountClientFactory:    input.BrokerAccountClientFactory,
		brokerOnBoardingClientFactory: input.BrokerOnBoardingClientFactory,
		backendRedirectURI:            input.BackendRedirectURI,
		frontendBaseURL:               input.FrontendBaseURL,
	}
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/accounts/v1/authorization_callback", handler.HandleAuthorizationCallback).Methods(http.MethodGet).Name("HandleAuthorizationCallback")

	accountV1Router := router.PathPrefix("/accounts/v1").Subrouter()
	accountV1Router.Use(input.AuthMiddleware.Handle)
	accountV1Router.HandleFunc("/accounts", handler.CreateAccount).Methods(http.MethodPost).Name("CreateAccount")
	accountV1Router.HandleFunc("/accounts", handler.ListAccounts).Methods(http.MethodGet).Name("ListAccounts")
	accountV1Router.HandleFunc("/accounts/{account_id}", handler.GetAccount).Methods(http.MethodGet).Name("GetAccount")
	accountV1Router.HandleFunc("/accounts/{account_id}/balances", handler.GetAccountBalance).Methods(http.MethodGet).Name("GetAccountBalance")
	accountV1Router.HandleFunc("/accounts/{account_id}/pnl/daily", handler.GetDailyPnL).Methods(http.MethodGet).Name("GetDailyPnL")

	accountV1Router.HandleFunc("/accounts/{account_id}/brokers", handler.StartBrokerSelection).Methods(http.MethodPost).Name("StartBrokerSelection")
	accountV1Router.HandleFunc("/accounts/{account_id}/brokers", handler.GetPendingBrokerSelection).Methods(http.MethodGet).Name("GetPendingBrokerSelection")
	accountV1Router.HandleFunc("/accounts/{account_id}/brokers", handler.CompleteBrokerSelection).Methods(http.MethodPut).Name("CompleteBrokerSelection")
	return router
}

func GenerateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

var merryErrorByAccountStoreError = map[error]error{
	accountstore.ErrAccountNotFound:            merry.New("account not found").WithHTTPCode(http.StatusNotFound),
	accountstore.ErrAccountForbidden:           merry.New("forbidden").WithHTTPCode(http.StatusForbidden),
	accountstore.ErrBrokerAccountAlreadyLinked: merry.New("broker already linked").WithHTTPCode(http.StatusConflict),
}
