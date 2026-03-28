package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/httputil"
)

type TastyTradeAuthorizationClient struct {
	backendRedirectURI string
	credentials        auth.Credentials
}

type TastyTradeAuthorizationClientInput struct {
	BackendRedirectURI string
	Credentials        auth.Credentials
}

func NewTastyTradeAuthorizationClient(input TastyTradeAuthorizationClientInput) *TastyTradeAuthorizationClient {
	return &TastyTradeAuthorizationClient{
		backendRedirectURI: input.BackendRedirectURI,
		credentials:        input.Credentials,
	}
}

func (client *TastyTradeAuthorizationClient) BuildAuthorizationURL(stateToken string) (string, error) {
	authURL, err := url.Parse(client.credentials.AuthorizationServer.AuthorizationEndpoint)
	if err != nil {
		panic(err)
	}
	authURL.RawQuery = url.Values{
		"response_type": {"code"},
		"client_id":     {client.credentials.AuthorizationServer.ClientCredentials.ClientID},
		"redirect_uri":  {client.backendRedirectURI},
		"state":         {stateToken},
	}.Encode()
	return authURL.String(), nil
}

type tastyTradeExchangeCodeRequestBody struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (client *TastyTradeAuthorizationClient) ExchangeCode(ctx context.Context, code string) (*AuthorizationTokens, error) {
	body, err := json.Marshal(tastyTradeExchangeCodeRequestBody{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  client.backendRedirectURI,
		ClientID:     client.credentials.AuthorizationServer.ClientCredentials.ClientID,
		ClientSecret: client.credentials.AuthorizationServer.ClientCredentials.ClientSecret,
	})
	if err != nil {
		return nil, merry.Wrap(err)
	}

	tokenURL, err := url.Parse(client.credentials.AuthorizationServer.TokenEndpoint)
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
		return nil, merry.Wrap(err)
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

	var output AuthorizationTokens
	if err = json.NewDecoder(response.Body).Decode(&output); err != nil {
		return nil, merry.Wrap(err)
	}
	return &output, nil
}

type tastyTradeCustomerAccountsResponse struct {
	Data struct {
		Items []struct {
			Account struct {
				AccountNumber string `json:"account-number"`
			} `json:"account"`
		} `json:"items"`
	} `json:"data"`
}

func (client *TastyTradeAuthorizationClient) ListAccounts(ctx context.Context, accessToken string) ([]string, error) {
	apiURL, err := url.Parse(client.credentials.APIURL)
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
		return nil, merry.Wrap(err)
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

	var payload tastyTradeCustomerAccountsResponse
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

func (client *TastyTradeAuthorizationClient) GenerateAccount(accountID string) (*Account, error) {
	if accountID == "" {
		return nil, merry.New("broker account id is required")
	}
	return &Account{
		Type: AccountTypeTastyTrade,
		TastyTrade: &AccountTastyTrade{
			ID: accountID,
		},
	}, nil
}
