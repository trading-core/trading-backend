package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) HandleAuthorizationCallback(responseWriter http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	frontendAccountURL := handler.frontendBaseURL + "/account"
	stateToken := request.URL.Query().Get("state")
	code := request.URL.Query().Get("code")
	if stateToken == "" || code == "" {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=missing_parameters", http.StatusFound)
		return
	}
	stateEntry, ok := handler.PopOAuthStateEntry(stateToken)
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
	pendingToken, err := GenerateStateToken()
	if err != nil {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=pending_token_failed", http.StatusFound)
		return
	}
	handler.PutPendingBrokerSelectionEntry(pendingToken, PendingBrokerSelectionEntry{
		AccountID:      stateEntry.AccountID,
		UserID:         stateEntry.UserID,
		BrokerAccounts: accountNumbers,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	})
	http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_pending="+url.QueryEscape(pendingToken)+"&oauth_account_id="+url.QueryEscape(stateEntry.AccountID), http.StatusFound)
}

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
