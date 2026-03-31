package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type GetPendingBrokerSelectionOutput struct {
	Broker         broker.AccountType `json:"broker"`
	BrokerAccounts []string           `json:"broker_accounts"`
}

func (handler *Handler) GetPendingBrokerSelection(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	pendingToken := request.URL.Query().Get("pending_token")
	if pendingToken == "" {
		err = merry.New("pending_token query parameter is required").WithHTTPCode(http.StatusBadRequest)
		return
	}
	entry, ok := handler.pendingSelectionStore.Get(pendingToken)
	if !ok {
		err = merry.New("pending broker selection not found").WithHTTPCode(http.StatusNotFound)
		return
	}
	if entry.UserID != userID {
		err = merry.New("forbidden").WithHTTPCode(http.StatusForbidden)
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(GetPendingBrokerSelectionOutput{
		Broker:         entry.Broker,
		BrokerAccounts: entry.BrokerAccounts,
	})
	fatal.OnErrorUnlessDone(ctx, err)
}
