package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/kduong/trading-backend/internal/contextx"
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
	accountID := contextx.GetAccountID(ctx)
	accountObject, err := handler.accountStore.Get(ctx, accountID)
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
