package httpapi

import (
	"net/http"
	"net/url"
	"time"

	"github.com/kduong/trading-backend/cmd/account-service/internal/pendingselectionstore"
	"github.com/kduong/trading-backend/internal/contextx"
)

func (handler *Handler) HandleAuthorizationCallback(responseWriter http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	frontendAccountURL := handler.frontendBaseURL + "/account"
	stateToken := request.URL.Query().Get("state")
	code := request.URL.Query().Get("code")
	if stateToken == "" || code == "" {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=missing_parameters", http.StatusFound)
		return
	}
	stateEntry, ok := handler.oauthStateStore.Pop(stateToken)
	if !ok {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=invalid_state", http.StatusFound)
		return
	}
	authorizationClient, err := handler.brokerOnBoardingClientFactory.GetAuthorizationClient(stateEntry.Broker)
	if err != nil {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=unsupported_broker", http.StatusFound)
		return
	}
	tokenOutput, err := authorizationClient.RequestAccessTokenUsingAuthorizationCode(ctx, code)
	if err != nil {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=token_exchange_failed", http.StatusFound)
		return
	}
	ctx = contextx.WithAccessToken(ctx, tokenOutput.AccessToken)
	accountDiscoveryClient, err := handler.brokerOnBoardingClientFactory.GetAccountDiscoveryClient(ctx, stateEntry.Broker)
	if err != nil {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=unsupported_broker", http.StatusFound)
		return
	}
	accountNumbers, err := accountDiscoveryClient.ListAccountIDs(ctx)
	if err != nil || len(accountNumbers) == 0 {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=no_accounts_found", http.StatusFound)
		return
	}
	pendingToken, err := GenerateStateToken()
	if err != nil {
		http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_error=pending_token_failed", http.StatusFound)
		return
	}
	handler.pendingSelectionStore.Put(pendingToken, pendingselectionstore.Entry{
		AccountID:      stateEntry.AccountID,
		UserID:         stateEntry.UserID,
		Broker:         stateEntry.Broker,
		BrokerAccounts: accountNumbers,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	})
	http.Redirect(responseWriter, request, frontendAccountURL+"?oauth_pending="+url.QueryEscape(pendingToken)+"&oauth_account_id="+url.QueryEscape(stateEntry.AccountID), http.StatusFound)
}
