package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) GetAccountBalance(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	accountID := vars["account_id"]
	account, err := handler.accountStore.Get(ctx, accountID)
	if err != nil {
		return
	}
	if !account.BrokerLinked {
		err = merry.New("account is not linked to a broker").WithHTTPCode(http.StatusBadRequest)
		return
	}
	broker := handler.brokerClientFactory.GetClient(ctx, account.BrokerAccount)
	balanceInfo, err := broker.GetBalanceInfo(ctx)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(balanceInfo)
	fatal.OnErrorUnlessDone(ctx, err)
}
