package broker

import (
	"bytes"
	"context"
	"encoding/json"
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

func (client *TastyTradeAuthorizationClient) BuildAuthorizationURL(stateToken string) string {
	authorizationURL, err := url.Parse(client.credentials.AuthorizationServer.AuthorizationEndpoint)
	if err != nil {
		panic(err)
	}
	authorizationQueryParams := url.Values{
		"response_type": {"code"},
		"client_id":     {client.credentials.AuthorizationServer.ClientCredentials.ClientID},
		"redirect_uri":  {client.backendRedirectURI},
		"state":         {stateToken},
	}
	authorizationURL.RawQuery = authorizationQueryParams.Encode()
	return authorizationURL.String()
}

type tastyTradeExchangeCodeRequestBody struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (client *TastyTradeAuthorizationClient) RequestAccessTokenUsingAuthorizationCode(ctx context.Context, code string) (output *TokenOutput, err error) {
	body, err := json.Marshal(tastyTradeExchangeCodeRequestBody{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  client.backendRedirectURI,
		ClientID:     client.credentials.AuthorizationServer.ClientCredentials.ClientID,
		ClientSecret: client.credentials.AuthorizationServer.ClientCredentials.ClientSecret,
	})
	if err != nil {
		return
	}
	target := client.credentials.AuthorizationServer.TokenEndpoint
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, merry.Wrap(err)
	}
	request.Header.Set("Content-Type", "application/json")
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
