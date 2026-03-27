package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type StartBrokerSelectionOutput struct {
	AuthorizationURL string `json:"authorization_url"`
}

func (handler *Handler) StartBrokerSelection(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	vars := mux.Vars(request)
	accountID := vars["account_id"]
	_, err = handler.accountStore.Get(ctx, accountstore.GetInput{
		AccountID: accountID,
	})
	if err != nil {
		err = merryErrorByAccountStoreError[err]
		return
	}
	stateToken, err := GenerateStateToken()
	if err != nil {
		return
	}
	handler.PutOAuthStateEntry(stateToken, OAuthStateEntry{
		AccountID: accountID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	authURL, err := url.Parse(handler.tastyTradeCredentials.AuthorizationServer.AuthorizationEndpoint)
	fatal.OnError(err)
	authURLQueryParameters := url.Values{
		"response_type": {"code"},
		"client_id":     {handler.tastyTradeCredentials.AuthorizationServer.ClientCredentials.ClientID},
		"redirect_uri":  {handler.backendRedirectURI},
		"state":         {stateToken},
	}
	authURL.RawQuery = authURLQueryParameters.Encode()
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(StartBrokerSelectionOutput{
		AuthorizationURL: authURL.String(),
	})
	fatal.OnErrorUnlessDone(ctx, err)
}
