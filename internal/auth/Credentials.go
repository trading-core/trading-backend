package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/fatal"
)

type Credentials struct {
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	APIURL              string                  `json:"api_url"`
	AuthorizationServer AuthorizationServerInfo `json:"authorization_server"`
}

func CredentialsByTypeFromEnv() map[string]Credentials {
	var credentialsByType map[string]Credentials
	data := config.EnvStringOrFatal("BROKER_CREDENTIALS_B64_JSON")
	reader := strings.NewReader(data)
	base64Decoder := base64.NewDecoder(base64.StdEncoding, reader)
	err := json.NewDecoder(base64Decoder).Decode(&credentialsByType)
	fatal.OnError(err)
	return credentialsByType
}

type AuthorizationServerInfo struct {
	AuthorizationEndpoint string            `json:"authorization_endpoint"`
	TokenEndpoint         string            `json:"token_endpoint"`
	ClientCredentials     ClientCredentials `json:"client_credentials"`
	RefreshToken          string            `json:"refresh_token"`
}

type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}
