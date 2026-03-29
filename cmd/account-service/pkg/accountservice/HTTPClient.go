package accountservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (client *HTTPClient) GetAccount(ctx context.Context, accountID string) (output *Account, err error) {
	target := url.URL{
		Scheme: client.baseURL.Scheme,
		Host:   client.baseURL.Host,
		Path:   fmt.Sprintf("/accounts/v1/accounts/%s", accountID),
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

func (client *HTTPClient) mapResponseError(response *http.Response) (err error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	message := strings.TrimSpace(string(body))
	switch response.StatusCode {
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrAccountForbidden, message)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrAccountNotFound, message)
	default:
		return fmt.Errorf("%w: %s", ErrServerError, message)
	}
}
