package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

var _ TokenManager = (*TastyTradeTokenManager)(nil)

type TastyTradeTokenManager struct {
	tokenURL     url.URL
	clientID     string
	clientSecret string
	refreshToken string

	mutex       sync.Mutex
	gracePeriod time.Duration
	accessToken string
	expiry      time.Time
}

func NewTastyTradeTokenManager(authorizationServerInfo *AuthorizationServerInfo) *TastyTradeTokenManager {
	tokenURL, err := url.Parse(authorizationServerInfo.TokenEndpoint)
	fatal.OnError(err)
	clientID := authorizationServerInfo.ClientCredentials.ClientID
	clientSecret := authorizationServerInfo.ClientCredentials.ClientSecret
	return &TastyTradeTokenManager{
		tokenURL:     *tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: authorizationServerInfo.RefreshToken,
		gracePeriod:  5 * time.Minute,
	}
}

func (tokenManager *TastyTradeTokenManager) GetAccessToken(ctx context.Context) (accessToken string, err error) {
	tokenManager.mutex.Lock()
	defer tokenManager.mutex.Unlock()
	if tokenManager.accessToken == "" || time.Now().After(tokenManager.expiry) {
		var output GetAccessTokenOutput
		output, err = tokenManager.RequestAccessToken(ctx)
		if err != nil {
			return
		}
		tokenManager.accessToken = output.AccessToken
		tokenManager.expiry = time.Now().Add(time.Duration(output.ExpiresIn) * time.Second).Add(-tokenManager.gracePeriod)
	}
	accessToken = tokenManager.accessToken
	return
}

type GetAccessTokenRequestBody struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type GetAccessTokenOutput struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

func (tokenManager *TastyTradeTokenManager) RequestAccessToken(ctx context.Context) (output GetAccessTokenOutput, err error) {
	target := url.URL{
		Scheme: tokenManager.tokenURL.Scheme,
		Host:   tokenManager.tokenURL.Host,
		Path:   "/oauth/token",
	}
	body, err := json.Marshal(GetAccessTokenRequestBody{
		GrantType:    "refresh_token",
		RefreshToken: tokenManager.refreshToken,
		ClientID:     tokenManager.clientID,
		ClientSecret: tokenManager.clientSecret,
	})
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		panic(err)
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
