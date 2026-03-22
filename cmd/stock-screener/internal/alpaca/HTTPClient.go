package alpaca

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kduong/trading-backend/internal/httputil"
)

var _ Client = (*HTTPClient)(nil)

type HTTPClient struct {
	*http.Client

	baseURL   url.URL
	keyID     string
	secretKey string
}

type NewHTTPClientInput struct {
	Timeout   time.Duration
	BaseURL   url.URL
	KeyID     string
	SecretKey string
}

func NewHTTPClient(input NewHTTPClientInput) *HTTPClient {
	httpClient := &http.Client{
		Timeout: input.Timeout,
	}
	return &HTTPClient{
		Client:    httpClient,
		baseURL:   input.BaseURL,
		keyID:     input.KeyID,
		secretKey: input.SecretKey,
	}
}

func (client *HTTPClient) GetActiveStocks(ctx context.Context, input GetActiveStocksInput) (output *GetActiveStocksOutput, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   "/v1beta1/screener/stocks/most-actives",
	}
	query := target.Query()
	query.Set("by", string(input.RankBy))
	query.Set("top", strconv.Itoa(input.Limit))
	target.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	request.Header.Set("APCA-API-KEY-ID", client.keyID)
	request.Header.Set("APCA-API-SECRET-KEY", client.secretKey)
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = client.extractResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) GetTopStockMovers(ctx context.Context, input GetTopStockMoversInput) (output *GetTopStockMoversOutput, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   "/v1beta1/screener/stocks/movers",
	}
	query := target.Query()
	query.Set("top", strconv.Itoa(input.Limit))
	target.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	request.Header.Set("APCA-API-KEY-ID", client.keyID)
	request.Header.Set("APCA-API-SECRET-KEY", client.secretKey)
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = client.extractResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) GetStockNews(ctx context.Context, input GetStockNewsInput) (output *GetStockNewsOutput, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   "/v1beta1/news",
	}
	query := target.Query()
	query.Set("limit", strconv.Itoa(input.Limit))
	if len(input.Symbols) > 0 {
		query.Set("symbols", strings.Join(input.Symbols, ","))
	}
	if input.PageToken != "" {
		query.Set("page_token", input.PageToken)
	}
	target.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	request.Header.Set("APCA-API-KEY-ID", client.keyID)
	request.Header.Set("APCA-API-SECRET-KEY", client.secretKey)
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = client.extractResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) extractResponseError(response *http.Response) error {
	data, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	return fmt.Errorf("failed to perform request: %s", string(data))
}
