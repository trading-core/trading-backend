package tastytrade

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kduong/trading-backend/internal/httputil"
)

type HTTPClient struct {
	apiURL         *url.URL
	getAccessToken func(ctx context.Context) (accessToken string, err error)
}

type NewHTTPClientInput struct {
	APIURL         *url.URL
	GetAccessToken func(ctx context.Context) (accessToken string, err error)
}

func NewHTTPClient(input NewHTTPClientInput) *HTTPClient {
	return &HTTPClient{
		apiURL:         input.APIURL,
		getAccessToken: input.GetAccessToken,
	}
}

func (client *HTTPClient) ListAccounts(ctx context.Context) (output []*Accounts, err error) {
	target := url.URL{
		Scheme: client.apiURL.Scheme,
		Host:   client.apiURL.Host,
		Path:   "/customers/me/accounts",
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	accessToken, err := client.getAccessToken(ctx)
	if err != nil {
		return
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httputil.ExtractResponseError(response)
		return
	}
	var accounts Accounts
	err = json.NewDecoder(response.Body).Decode(&accounts)
	if err != nil {
		return
	}
	output = []*Accounts{&accounts}
	return
}

func (client *HTTPClient) GetAccountBalance(ctx context.Context, accountID string) (output *AccountBalance, err error) {
	target := url.URL{
		Scheme: client.apiURL.Scheme,
		Host:   client.apiURL.Host,
		Path:   fmt.Sprintf("/accounts/%s/balances", accountID),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	accessToken, err := client.getAccessToken(ctx)
	if err != nil {
		return
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httputil.ExtractResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

type HTTPClientFactory struct {
	APIURL         *url.URL
	GetAccessToken func(ctx context.Context) (accessToken string, err error)
}

func (factory *HTTPClientFactory) Create() Client {
	return NewHTTPClient(NewHTTPClientInput{
		APIURL:         factory.APIURL,
		GetAccessToken: factory.GetAccessToken,
	})
}
