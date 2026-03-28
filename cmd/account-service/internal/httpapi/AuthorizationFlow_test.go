package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/contextx"
)

type stubAccountStore struct {
	getAccount       *accountstore.Account
	linkAccountInput accountstore.LinkBrokerAccountInput
	linkCalls        int
}

func (store *stubAccountStore) Create(ctx context.Context, input accountstore.CreateInput) error {
	panic("unexpected call to Create")
}

func (store *stubAccountStore) LinkBrokerAccount(ctx context.Context, input accountstore.LinkBrokerAccountInput) error {
	store.linkAccountInput = input
	store.linkCalls++
	return nil
}

func (store *stubAccountStore) Get(ctx context.Context, input accountstore.GetInput) (*accountstore.Account, error) {
	return store.getAccount, nil
}

func (store *stubAccountStore) List(ctx context.Context) ([]*accountstore.Account, error) {
	panic("unexpected call to List")
}

func newTestHandler(store accountstore.Store, serverURL string) *Handler {
	credentials := auth.Credentials{
		APIURL: serverURL,
		AuthorizationServer: auth.AuthorizationServerInfo{
			AuthorizationEndpoint: serverURL + "/oauth/authorize",
			TokenEndpoint:         serverURL + "/oauth/token-proxy",
			ClientCredentials: auth.ClientCredentials{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
	}
	return &Handler{
		oauthStateStore:       map[string]OAuthStateEntry{},
		pendingSelectionStore: map[string]PendingBrokerSelectionEntry{},
		accountStore:          store,
		brokerAuthorizationFactory: &broker.AuthorizationClientFactory{
			BackendRedirectURI: "http://backend.example/accounts/v1/authorization_callback",
			CredentialsByType: map[broker.AccountType]auth.Credentials{
				broker.AccountTypeTastyTrade: credentials,
			},
		},
		frontendBaseURL: "http://frontend.example",
	}
}

func TestStartBrokerSelectionPersistsBrokerType(t *testing.T) {
	store := &stubAccountStore{
		getAccount: &accountstore.Account{ID: "acct-1", UserID: "user-1"},
	}
	handler := newTestHandler(store, "http://broker.example")
	body := strings.NewReader(`{"broker":"tastytrade"}`)
	req := httptest.NewRequest(http.MethodPost, "/accounts/v1/accounts/acct-1/brokers", body)
	req = mux.SetURLVars(req, map[string]string{"account_id": "acct-1"})
	req = req.WithContext(contextx.WithUserID(req.Context(), "user-1"))
	recorder := httptest.NewRecorder()

	handler.StartBrokerSelection(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if len(handler.oauthStateStore) != 1 {
		t.Fatalf("expected 1 oauth state entry, got %d", len(handler.oauthStateStore))
	}
	for _, entry := range handler.oauthStateStore {
		if entry.Broker != broker.AccountTypeTastyTrade {
			t.Fatalf("expected broker type tastytrade, got %s", entry.Broker)
		}
	}
	var output StartBrokerSelectionOutput
	if err := json.NewDecoder(recorder.Body).Decode(&output); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	parsedURL, err := url.Parse(output.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse authorization url: %v", err)
	}
	if parsedURL.Query().Get("client_id") != "client-id" {
		t.Fatalf("expected client_id query param to be preserved")
	}
}

func TestAuthorizationCallbackPersistsPendingBrokerType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/oauth/token":
			responseWriter.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(responseWriter, `{"access_token":"token-123","expires_in":3600,"token_type":"Bearer"}`)
		case "/customers/me/accounts":
			responseWriter.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(responseWriter, `{"data":{"items":[{"account":{"account-number":"broker-1"}},{"account":{"account-number":"broker-2"}}]}}`)
		default:
			http.NotFound(responseWriter, request)
		}
	}))
	defer server.Close()

	handler := newTestHandler(&stubAccountStore{}, server.URL)
	handler.PutOAuthStateEntry("state-123", OAuthStateEntry{
		AccountID: "acct-1",
		UserID:    "user-1",
		Broker:    broker.AccountTypeTastyTrade,
		ExpiresAt: time.Now().Add(time.Minute),
	})
	req := httptest.NewRequest(http.MethodGet, "/accounts/v1/authorization_callback?state=state-123&code=auth-code", nil)
	recorder := httptest.NewRecorder()

	handler.HandleAuthorizationCallback(recorder, req)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", recorder.Code)
	}
	if len(handler.pendingSelectionStore) != 1 {
		t.Fatalf("expected 1 pending selection entry, got %d", len(handler.pendingSelectionStore))
	}
	for _, entry := range handler.pendingSelectionStore {
		if entry.Broker != broker.AccountTypeTastyTrade {
			t.Fatalf("expected pending broker type tastytrade, got %s", entry.Broker)
		}
		if len(entry.BrokerAccounts) != 2 {
			t.Fatalf("expected 2 broker accounts, got %d", len(entry.BrokerAccounts))
		}
	}
}

func TestCompleteBrokerSelectionUsesPendingBrokerType(t *testing.T) {
	store := &stubAccountStore{}
	handler := newTestHandler(store, "http://broker.example")
	handler.PutPendingBrokerSelectionEntry("pending-123", PendingBrokerSelectionEntry{
		AccountID:      "acct-1",
		UserID:         "user-1",
		Broker:         broker.AccountTypeTastyTrade,
		BrokerAccounts: []string{"broker-1"},
		ExpiresAt:      time.Now().Add(time.Minute),
	})
	body := strings.NewReader(`{"pending_token":"pending-123","broker_account_id":"broker-1"}`)
	req := httptest.NewRequest(http.MethodPut, "/accounts/v1/accounts/acct-1/brokers", body)
	req = req.WithContext(contextx.WithUserID(req.Context(), "user-1"))
	recorder := httptest.NewRecorder()

	handler.CompleteBrokerSelection(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if store.linkCalls != 1 {
		t.Fatalf("expected link to be called once, got %d", store.linkCalls)
	}
	if store.linkAccountInput.BrokerAccount == nil {
		t.Fatal("expected broker account to be linked")
	}
	if store.linkAccountInput.BrokerAccount.Type != broker.AccountTypeTastyTrade {
		t.Fatalf("expected tastytrade broker account type, got %s", store.linkAccountInput.BrokerAccount.Type)
	}
	if store.linkAccountInput.BrokerAccount.TastyTrade == nil || store.linkAccountInput.BrokerAccount.TastyTrade.ID != "broker-1" {
		t.Fatal("expected tastytrade broker account id to be preserved")
	}
}
