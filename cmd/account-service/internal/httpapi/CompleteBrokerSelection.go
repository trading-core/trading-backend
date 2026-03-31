package httpapi

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type CompleteBrokerSelectionInput struct {
	PendingToken    string `json:"pending_token"`
	BrokerAccountID string `json:"broker_account_id"`
}

type CompleteBrokerSelectionOutput struct {
	AccountID     string         `json:"account_id"`
	BrokerAccount broker.Account `json:"broker_account"`
}

func (handler *Handler) CompleteBrokerSelection(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	var input CompleteBrokerSelectionInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	if input.PendingToken == "" || input.BrokerAccountID == "" {
		err = merry.New("pending_token and broker_account_id are required").WithHTTPCode(http.StatusBadRequest)
		return
	}
	entry, ok := handler.pendingSelectionStore.Get(input.PendingToken)
	if !ok {
		err = merry.New("pending broker selection not found").WithHTTPCode(http.StatusNotFound)
		return
	}
	if entry.UserID != userID {
		err = merry.New("forbidden").WithHTTPCode(http.StatusForbidden)
		return
	}
	isValidBrokerAccount := slices.Contains(entry.BrokerAccounts, input.BrokerAccountID)
	if !isValidBrokerAccount {
		err = merry.New("broker account is not available for this selection").WithHTTPCode(http.StatusBadRequest)
		return
	}
	brokerAccount := &broker.Account{
		Type: entry.Broker,
		ID:   input.BrokerAccountID,
	}
	ctx = contextx.WithUserID(ctx, entry.UserID)
	err = handler.accountStoreCommandHandler.LinkBrokerAccount(ctx, accountstore.LinkBrokerAccountInput{
		AccountID:     entry.AccountID,
		BrokerAccount: brokerAccount,
	})
	if err != nil {
		err = merryErrorByAccountStoreError[err]
		return
	}
	handler.pendingSelectionStore.Delete(input.PendingToken)
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(CompleteBrokerSelectionOutput{
		AccountID:     entry.AccountID,
		BrokerAccount: *brokerAccount,
	})
	fatal.OnErrorUnlessDone(ctx, err)
}
