package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type StartBrokerSelectionInput struct {
	Broker broker.AccountType `json:"broker"`
}

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
	var input StartBrokerSelectionInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	if input.Broker == "" {
		err = merry.New("broker is required").WithHTTPCode(http.StatusBadRequest)
		return
	}
	_, err = handler.accountStoreQueryHandler.Get(ctx, accountstore.GetInput{
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
	authorizationClient, err := handler.brokerOnBoardingClientFactory.GetAuthorizationClient(input.Broker)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest)
		return
	}
	handler.PutOAuthStateEntry(stateToken, OAuthStateEntry{
		AccountID: accountID,
		UserID:    userID,
		Broker:    input.Broker,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(StartBrokerSelectionOutput{
		AuthorizationURL: authorizationClient.BuildAuthorizationURL(stateToken),
	})
	fatal.OnErrorUnlessDone(ctx, err)
}
