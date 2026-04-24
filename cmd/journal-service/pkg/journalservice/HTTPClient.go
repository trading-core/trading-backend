package journalservice

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

	"github.com/kduong/trading-backend/internal/contextx"
)

type HTTPClient struct {
	baseURL    url.URL
	httpClient *http.Client
}

type NewHTTPClientInput struct {
	Timeout time.Duration
	BaseURL url.URL
}

func NewHTTPClient(input NewHTTPClientInput) *HTTPClient {
	return &HTTPClient{
		baseURL: input.BaseURL,
		httpClient: &http.Client{
			Timeout: input.Timeout,
		},
	}
}

func (client *HTTPClient) GetEntry(ctx context.Context, date string) (output *Entry, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   fmt.Sprintf("/journal/v1/entries/%s", date),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	accessToken := contextx.GetAccessToken(ctx)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = client.mapResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) ListEntries(ctx context.Context, input ListInput) (output *ListResult, err error) {
	query := url.Values{}
	if input.From != "" {
		query.Set("from", input.From)
	}
	if input.To != "" {
		query.Set("to", input.To)
	}
	if input.Page != 0 {
		query.Set("page", strconv.Itoa(input.Page))
	}
	if input.PageSize != 0 {
		query.Set("page_size", strconv.Itoa(input.PageSize))
	}
	target := url.URL{
		Scheme:   client.baseURL.Scheme,
		Host:     client.baseURL.Host,
		Path:     "/journal/v1/entries",
		RawQuery: query.Encode(),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		panic(err)
	}
	accessToken := contextx.GetAccessToken(ctx)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = client.mapResponseError(response)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *HTTPClient) mapResponseError(response *http.Response) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	message := strings.TrimSpace(string(body))
	switch response.StatusCode {
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrEntryForbidden, message)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrEntryNotFound, message)
	default:
		return fmt.Errorf("%w: %s", ErrServerError, message)
	}
}
