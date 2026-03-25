package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
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
	account, err := handler.accountStore.Get(ctx, accountID)
	if err != nil {
		return
	}
	if account.BrokerAccount == nil {
		err = merry.New("no linked broker account found").WithHTTPCode(http.StatusBadRequest)
		return
	}
	broker := handler.brokerAdapterFactory.GetBrokerAdapter(ctx, account.BrokerAccount)
	balanceInfo, err := broker.GetBalanceInfo(ctx)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(balanceInfo)
	fatal.OnErrorUnlessDone(ctx, err)
}
