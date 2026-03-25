package auth

type Credentials struct {
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	APIURL              string                  `json:"api_url"`
	AuthorizationServer AuthorizationServerInfo `json:"authorization_server"`
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
