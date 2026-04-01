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
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
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
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
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

type SearchSymbolsResponse struct {
	Data SearchSymbolsData `json:"data"`
}

type SearchSymbolsData struct {
	Items []*Symbol `json:"items"`
}

func (client *HTTPClient) SearchSymbol(ctx context.Context, symbol string) (output *Symbol, err error) {
	target := url.URL{
		Scheme: client.apiURL.Scheme,
		Host:   client.apiURL.Host,
		Path:   fmt.Sprintf("/symbols/search/%s", symbol),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httputil.ExtractResponseError(response)
		return
	}
	var body SearchSymbolsResponse
	err = json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		return
	}
	if len(body.Data.Items) == 0 {
		err = ErrSymbolNotFound
		return
	}
	output = body.Data.Items[0]
	return
}

func (client *HTTPClient) GetAPIQuoteToken(ctx context.Context) (output *GetAPIQuoteTokenOutput, err error) {
	target := url.URL{
		Scheme: client.apiURL.Scheme,
		Host:   client.apiURL.Host,
		Path:   "/api-quote-tokens",
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	accessToken, err := client.getAccessToken(ctx)
	if err != nil {
		return
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
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
