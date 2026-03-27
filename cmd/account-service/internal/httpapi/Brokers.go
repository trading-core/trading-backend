package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

// --- In-memory state store (CSRF protection) ----------------------------

type oauthStateEntry struct {
	AccountID string
	UserID    string
	ExpiresAt time.Time
}

type pendingBrokerSelectionEntry struct {
	AccountID      string
	UserID         string
	BrokerAccounts []string
	ExpiresAt      time.Time
}

var (
	oauthStateMu          sync.Mutex
	oauthStateStore       = map[string]oauthStateEntry{}
	pendingSelectionMu    sync.Mutex
	pendingSelectionStore = map[string]pendingBrokerSelectionEntry{}
)

func generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func putOAuthState(token string, entry oauthStateEntry) {
	oauthStateMu.Lock()
	defer oauthStateMu.Unlock()
	// Purge any expired entries to prevent unbounded growth.
	for k, v := range oauthStateStore {
		if time.Now().After(v.ExpiresAt) {
			delete(oauthStateStore, k)
		}
	}
	oauthStateStore[token] = entry
}

func putPendingBrokerSelection(token string, entry pendingBrokerSelectionEntry) {
	pendingSelectionMu.Lock()
	defer pendingSelectionMu.Unlock()
	for k, v := range pendingSelectionStore {
		if time.Now().After(v.ExpiresAt) {
			delete(pendingSelectionStore, k)
		}
	}
	pendingSelectionStore[token] = entry
}

// popOAuthState removes and returns the entry for token, returning false if the
// token is unknown or expired.
func popOAuthState(token string) (oauthStateEntry, bool) {
	oauthStateMu.Lock()
	defer oauthStateMu.Unlock()
	entry, ok := oauthStateStore[token]
	if !ok {
		return oauthStateEntry{}, false
	}
	delete(oauthStateStore, token)
	if time.Now().After(entry.ExpiresAt) {
		return oauthStateEntry{}, false
	}
	return entry, true
}

func getPendingBrokerSelection(token string) (pendingBrokerSelectionEntry, bool) {
	pendingSelectionMu.Lock()
	defer pendingSelectionMu.Unlock()
	entry, ok := pendingSelectionStore[token]
	if !ok {
		return pendingBrokerSelectionEntry{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(pendingSelectionStore, token)
		return pendingBrokerSelectionEntry{}, false
	}
	return entry, true
}

func deletePendingBrokerSelection(token string) {
	pendingSelectionMu.Lock()
	defer pendingSelectionMu.Unlock()
	delete(pendingSelectionStore, token)
}

// --- Start handler -------------------------------------------------------

type startOAuth2Output struct {
	AuthorizationURL string `json:"authorization_url"`
}

type getPendingBrokerSelectionOutput struct {
	BrokerAccounts []string `json:"broker_accounts"`
}

type completeBrokerSelectionInput struct {
	PendingToken    string `json:"pending_token"`
	BrokerAccountID string `json:"broker_account_id"`
}

type completeBrokerSelectionOutput struct {
	AccountID     string         `json:"account_id"`
	BrokerAccount broker.Account `json:"broker_account"`
}

// StartOAuth2TastyTrade builds the Tastytrade authorization URL and returns it
// so the client can redirect the browser. Requires a valid JWT (auth middleware
// is applied to this route).
func (handler *Handler) StartBrokerSelection(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	accountID := mux.Vars(request)["account_id"]
	account, err := handler.accountStore.Get(ctx, account.GetInput{
		AccountID: accountID,
	})
	if err != nil {
		return
	}
	if account.UserID != userID {
		err = merry.New("forbidden").WithHTTPCode(http.StatusForbidden)
		return
	}
	if account.BrokerLinked {
		err = merry.New("broker already linked").WithHTTPCode(http.StatusConflict)
		return
	}
	stateToken, err := generateStateToken()
	if err != nil {
		return
	}
	putOAuthState(stateToken, oauthStateEntry{
		AccountID: accountID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	authURL, err := url.Parse(handler.tastyTradeCredentials.AuthorizationServer.AuthorizationEndpoint)
	if err != nil {
		err = merry.Wrap(err)
		return
	}
	authURL.RawQuery = url.Values{
		"response_type": {"code"},
		"client_id":     {handler.tastyTradeCredentials.AuthorizationServer.ClientCredentials.ClientID},
		"redirect_uri":  {handler.backendRedirectURI},
		"state":         {stateToken},
	}.Encode()

	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(startOAuth2Output{
		AuthorizationURL: authURL.String(),
	})
	fatal.OnErrorUnlessDone(ctx, err)
}

// --- Callback handler ---------------------------------------------------

type exchangeCodeRequestBody struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type exchangeCodeResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func (handler *Handler) exchangeCodeForToken(ctx context.Context, code string) (*exchangeCodeResponse, error) {
	body, err := json.Marshal(exchangeCodeRequestBody{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  handler.backendRedirectURI,
		ClientID:     handler.tastyTradeCredentials.AuthorizationServer.ClientCredentials.ClientID,
		ClientSecret: handler.tastyTradeCredentials.AuthorizationServer.ClientCredentials.ClientSecret,
	})
	if err != nil {
		panic(err)
	}

	tokenURL, err := url.Parse(handler.tastyTradeCredentials.AuthorizationServer.TokenEndpoint)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	target := url.URL{
		Scheme: tokenURL.Scheme,
		Host:   tokenURL.Host,
		Path:   "/oauth/token",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		return nil, merry.Wrap(httputil.ExtractResponseError(response))
	}

	var output exchangeCodeResponse
	if err = json.NewDecoder(response.Body).Decode(&output); err != nil {
		return nil, merry.Wrap(err)
	}
	return &output, nil
}

type customerAccountsResponse struct {
	Data struct {
		Items []struct {
			Account struct {
				AccountNumber string `json:"account-number"`
			} `json:"account"`
		} `json:"items"`
	} `json:"data"`
}

func (handler *Handler) getCustomerAccounts(ctx context.Context, accessToken string) ([]string, error) {
	apiURL, err := url.Parse(handler.tastyTradeCredentials.APIURL)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	target := url.URL{
		Scheme: apiURL.Scheme,
		Host:   apiURL.Host,
		Path:   "/customers/me/accounts",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		return nil, merry.Wrap(httputil.ExtractResponseError(response))
	}

	var payload customerAccountsResponse
	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, merry.Wrap(err)
	}

	accountNumbers := make([]string, 0, len(payload.Data.Items))
	for _, item := range payload.Data.Items {
		if item.Account.AccountNumber != "" {
			accountNumbers = append(accountNumbers, item.Account.AccountNumber)
		}
	}
	return accountNumbers, nil
}

// HandleAuthorizationCallback handles the redirect from Tastytrade after the user
// authorizes. It validates the state token, exchanges the code for a user
// access token, retrieves the user's Tastytrade accounts, and links the first
// account to the trading account stored in the state. This route is
// unauthenticated — the user identity is embedded in the signed state token.
func (handler *Handler) HandleAuthorizationCallback(responseWriter http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	frontendAccountURL := handler.frontendBaseURL + "/account"

	stateToken := request.URL.Query().Get("state")
	code := request.URL.Query().Get("code")
	if stateToken == "" || code == "" {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=missing_parameters", http.StatusFound)
		return
	}

	stateEntry, ok := popOAuthState(stateToken)
	if !ok {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=invalid_state", http.StatusFound)
		return
	}

	tokens, err := handler.exchangeCodeForToken(ctx, code)
	if err != nil {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=token_exchange_failed", http.StatusFound)
		return
	}

	accountNumbers, err := handler.getCustomerAccounts(ctx, tokens.AccessToken)
	if err != nil || len(accountNumbers) == 0 {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=no_accounts_found", http.StatusFound)
		return
	}

	pendingToken, err := generateStateToken()
	if err != nil {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=pending_token_failed", http.StatusFound)
		return
	}
	putPendingBrokerSelection(pendingToken, pendingBrokerSelectionEntry{
		AccountID:      stateEntry.AccountID,
		UserID:         stateEntry.UserID,
		BrokerAccounts: accountNumbers,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	})

	http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_pending="+url.QueryEscape(pendingToken)+"&oauth_account_id="+url.QueryEscape(stateEntry.AccountID), http.StatusFound)
}

func (handler *Handler) GetPendingBrokerSelection(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	pendingToken := request.URL.Query().Get("pending_token")
	if pendingToken == "" {
		err = merry.New("pending_token query parameter is required").WithHTTPCode(http.StatusBadRequest)
		return
	}
	entry, ok := getPendingBrokerSelection(pendingToken)
	if !ok {
		err = merry.New("pending broker selection not found").WithHTTPCode(http.StatusNotFound)
		return
	}
	if entry.UserID != userID {
		err = merry.New("forbidden").WithHTTPCode(http.StatusForbidden)
		return
	}
	httputil.SendResponseJSON(responseWriter, http.StatusOK, getPendingBrokerSelectionOutput{
		BrokerAccounts: entry.BrokerAccounts,
	})
}

func (handler *Handler) CompleteBrokerSelection(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	var input completeBrokerSelectionInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	if input.PendingToken == "" || input.BrokerAccountID == "" {
		err = merry.New("pending_token and broker_account_id are required").WithHTTPCode(http.StatusBadRequest)
		return
	}
	entry, ok := getPendingBrokerSelection(input.PendingToken)
	if !ok {
		err = merry.New("pending broker selection not found").WithHTTPCode(http.StatusNotFound)
		return
	}
	if entry.UserID != userID {
		err = merry.New("forbidden").WithHTTPCode(http.StatusForbidden)
		return
	}
	isValidBrokerAccount := false
	for _, brokerAccountID := range entry.BrokerAccounts {
		if brokerAccountID == input.BrokerAccountID {
			isValidBrokerAccount = true
			break
		}
	}
	if !isValidBrokerAccount {
		err = merry.New("broker account is not available for this selection").WithHTTPCode(http.StatusBadRequest)
		return
	}
	brokerAccount := &broker.Account{
		Type: broker.AccountTypeTastyTrade,
		TastyTrade: &broker.AccountTastyTrade{
			ID: input.BrokerAccountID,
		},
	}
	ctx = contextx.WithUserID(ctx, entry.UserID)
	err = handler.accountStore.LinkBrokerAccount(ctx, account.LinkBrokerAccountInput{
		AccountID:     entry.AccountID,
		BrokerAccount: brokerAccount,
	})
	if err != nil {
		return
	}
	deletePendingBrokerSelection(input.PendingToken)
	httputil.SendResponseJSON(responseWriter, http.StatusOK, completeBrokerSelectionOutput{
		AccountID:     entry.AccountID,
		BrokerAccount: *brokerAccount,
	})
}
