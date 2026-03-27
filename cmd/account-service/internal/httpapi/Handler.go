package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
)

type Handler struct {
	oauthStateMutex       sync.Mutex
	oauthStateStore       map[string]OAuthStateEntry
	pendingSelectionMutex sync.Mutex
	pendingSelectionStore map[string]PendingBrokerSelectionEntry

	accountStore          accountstore.Store
	brokerClientFactory   *broker.ClientFactory
	backendRedirectURI    string
	tastyTradeCredentials auth.Credentials
	frontendBaseURL       string
}

type NewRouterInput struct {
	AccountStore          accountstore.Store
	BrokerClientFactory   *broker.ClientFactory
	AuthMiddleWare        *auth.MiddleWare
	BackendRedirectURI    string
	TastyTradeCredentials auth.Credentials
	FrontendBaseURL       string
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		oauthStateStore:       make(map[string]OAuthStateEntry),
		pendingSelectionStore: make(map[string]PendingBrokerSelectionEntry),
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

type OAuthStateEntry struct {
	AccountID string
	UserID    string
	ExpiresAt time.Time
}

func (handler *Handler) PutOAuthStateEntry(token string, entry OAuthStateEntry) {
	handler.oauthStateMutex.Lock()
	defer handler.oauthStateMutex.Unlock()
	for key, value := range handler.oauthStateStore {
		if time.Now().After(value.ExpiresAt) {
			delete(handler.oauthStateStore, key)
		}
	}
	handler.oauthStateStore[token] = entry
}

func (handler *Handler) PopOAuthStateEntry(token string) (OAuthStateEntry, bool) {
	handler.oauthStateMutex.Lock()
	defer handler.oauthStateMutex.Unlock()
	entry, ok := handler.oauthStateStore[token]
	if !ok {
		return OAuthStateEntry{}, false
	}
	delete(handler.oauthStateStore, token)
	if time.Now().After(entry.ExpiresAt) {
		return OAuthStateEntry{}, false
	}
	return entry, true
}

type PendingBrokerSelectionEntry struct {
	AccountID      string
	UserID         string
	BrokerAccounts []string
	ExpiresAt      time.Time
}

func (handler *Handler) PutPendingBrokerSelectionEntry(token string, entry PendingBrokerSelectionEntry) {
	handler.pendingSelectionMutex.Lock()
	defer handler.pendingSelectionMutex.Unlock()
	for k, v := range handler.pendingSelectionStore {
		if time.Now().After(v.ExpiresAt) {
			delete(handler.pendingSelectionStore, k)
		}
	}
	handler.pendingSelectionStore[token] = entry
}

func (handler *Handler) GetPendingBrokerSelectionEntry(token string) (PendingBrokerSelectionEntry, bool) {
	handler.pendingSelectionMutex.Lock()
	defer handler.pendingSelectionMutex.Unlock()
	entry, ok := handler.pendingSelectionStore[token]
	if !ok {
		return PendingBrokerSelectionEntry{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(handler.pendingSelectionStore, token)
		return PendingBrokerSelectionEntry{}, false
	}
	return entry, true
}

func (handler *Handler) DeletePendingBrokerSelectionEntry(token string) {
	handler.pendingSelectionMutex.Lock()
	defer handler.pendingSelectionMutex.Unlock()
	delete(handler.pendingSelectionStore, token)
}

func GenerateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

var merryErrorByAccountStoreError = map[error]error{
	accountstore.ErrNotFound:                   merry.New("account not found").WithHTTPCode(http.StatusNotFound),
	accountstore.ErrForbidden:                  merry.New("forbidden").WithHTTPCode(http.StatusForbidden),
	accountstore.ErrBrokerAccountAlreadyLinked: merry.New("broker already linked").WithHTTPCode(http.StatusConflict),
}
