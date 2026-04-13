package tastytrade

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kduong/trading-backend/internal/httpx"
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
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httpx.ExtractResponseError(response)
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
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httpx.ExtractResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) GetAccountPositions(ctx context.Context, accountID string) (output *AccountPositionsOutput, err error) {
	target := url.URL{
		Scheme: client.apiURL.Scheme,
		Host:   client.apiURL.Host,
		Path:   fmt.Sprintf("/accounts/%s/positions", accountID),
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
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httpx.ExtractResponseError(response)
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
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httpx.ExtractResponseError(response)
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
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httpx.ExtractResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

type orderLeg struct {
	Action         string  `json:"action"`
	InstrumentType string  `json:"instrument-type"`
	Symbol         string  `json:"symbol"`
	Quantity       float64 `json:"quantity"`
}

type orderRequest struct {
	OrderType   string     `json:"order-type"`
	TimeInForce string     `json:"time-in-force"`
	Legs        []orderLeg `json:"legs"`
}

type order struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type orderData struct {
	Order order `json:"order"`
}

type orderResponse struct {
	Data orderData `json:"data"`
}

func (client *HTTPClient) PlaceEquityOrder(ctx context.Context, input PlaceEquityOrderInput) (output *PlaceEquityOrderOutput, err error) {
	target := url.URL{
		Scheme: client.apiURL.Scheme,
		Host:   client.apiURL.Host,
		Path:   fmt.Sprintf("/accounts/%s/orders", input.AccountID),
	}
	body := orderRequest{
		OrderType:   "Market",
		TimeInForce: "Day",
		Legs: []orderLeg{{
			Action:         input.Action,
			InstrumentType: "Equity",
			Symbol:         input.Symbol,
			Quantity:       input.Quantity,
		}},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		panic(err)
	}
	accessToken, err := client.getAccessToken(ctx)
	if err != nil {
		return
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		err = httpx.ExtractResponseError(response)
		return
	}
	var resp orderResponse
	err = json.NewDecoder(response.Body).Decode(&resp)
	if err != nil {
		return
	}
	output = &PlaceEquityOrderOutput{
		OrderID: resp.Data.Order.ID,
		Status:  resp.Data.Order.Status,
	}
	return
}

func (client *HTTPClient) GetLiveOrders(ctx context.Context, accountID string) (output *LiveOrdersOutput, err error) {
	target := url.URL{
		Scheme: client.apiURL.Scheme,
		Host:   client.apiURL.Host,
		Path:   fmt.Sprintf("/accounts/%s/orders/live", accountID),
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
	defer httpx.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httpx.ExtractResponseError(response)
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
