package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kduong/trading-backend/internal/account"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) GetBalance(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	accountID, err := handler.extractAccountID(request)
	if err != nil {
		return
	}
	accountObject, err := handler.accountStore.Get(ctx, accountID)
	if errors.Is(err, account.ErrAccountNotFound) {
		err = handler.accountStore.Put(ctx, &account.Object{
			AccountID:       accountID,
			BrokerType:      handler.defaultBrokerType,
			BrokerAccountID: handler.defaultBrokerAccountID,
		})
		if err != nil {
			return
		}
		accountObject, err = handler.accountStore.Get(ctx, accountID)
	}
	if err != nil {
		return
	}
	broker := handler.brokerAdapterFactory.GetBrokerAdapter(ctx, accountObject)
	balanceInfo, err := broker.GetBalanceInfo(ctx)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(balanceInfo)
	fatal.OnErrorUnlessDone(ctx, err)
}
