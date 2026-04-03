package alpaca

import (
	"context"
	"encoding/json"
	"fmt"
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
	client.authorizeRequest(request)
	response, err := client.Do(request)
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
	client.authorizeRequest(request)
	response, err := client.Do(request)
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

func (client *HTTPClient) GetStockSnapshot(ctx context.Context, input GetStockSnapshotInput) (output *GetStockSnapshotOutput, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   fmt.Sprintf("/v2/stocks/%s/snapshot", input.Symbol),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	client.authorizeRequest(request)
	response, err := client.Do(request)
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

type GetStockBarsResponseBody struct {
	Bars          []StockBarResponseBody `json:"bars"`
	NextPageToken string                 `json:"next_page_token"`
}

type StockBarResponseBody struct {
	Time  string  `json:"t"`
	Close float64 `json:"c"`
}

func (client *HTTPClient) GetStockBars(ctx context.Context, input GetStockBarsInput) (output *GetStockBarsOutput, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   fmt.Sprintf("/v2/stocks/%s/bars", input.Symbol),
	}
	query := buildStockBarsQuery(input)
	var allStockBars []StockBar
	var pageToken string
	for {
		stockBars, nextToken, fetchErr := client.fetchStockBarsPage(ctx, target, query, pageToken)
		if fetchErr != nil {
			err = fetchErr
			return
		}
		allStockBars = append(allStockBars, stockBars...)
		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}
	output = &GetStockBarsOutput{
		Bars: allStockBars,
	}
	return
}

func buildStockBarsQuery(input GetStockBarsInput) url.Values {
	query := make(url.Values)
	if input.Timeframe != "" {
		query.Set("timeframe", input.Timeframe)
	}
	if input.Limit > 0 {
		query.Set("limit", strconv.Itoa(input.Limit))
	}
	if input.Start != "" {
		query.Set("start", input.Start)
	}
	if input.End != "" {
		query.Set("end", input.End)
	}
	query.Set("adjustment", "raw")
	query.Set("sort", "asc")
	if input.Feed != "" {
		query.Set("feed", input.Feed)
	}
	return query
}

func (client *HTTPClient) fetchStockBarsPage(ctx context.Context, target url.URL, query url.Values, pageToken string) (bars []StockBar, nextToken string, err error) {
	q := query
	if pageToken != "" {
		q = make(url.Values)
		for k, v := range query {
			q[k] = v
		}
		q.Set("page_token", pageToken)
	}
	target.RawQuery = q.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return
	}
	client.authorizeRequest(request)
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer httputil.DrainAndClose(response.Body)
	if response.StatusCode != http.StatusOK {
		err = httputil.ExtractResponseError(response)
		return
	}
	var responseBody GetStockBarsResponseBody
	if err = json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return
	}
	for _, bar := range responseBody.Bars {
		bars = append(bars, StockBar{
			Time:  bar.Time,
			Close: bar.Close,
		})
	}
	nextToken = responseBody.NextPageToken
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
	client.authorizeRequest(request)
	response, err := client.Do(request)
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

func (client *HTTPClient) authorizeRequest(request *http.Request) {
	request.Header.Set("APCA-API-KEY-ID", client.keyID)
	request.Header.Set("APCA-API-SECRET-KEY", client.secretKey)
}
