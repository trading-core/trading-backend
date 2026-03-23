package httpapi

import (
	"encoding/json"
	"net/http"

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
	// TODO: extract account ID from request (e.g. from JWT claims) and pass it to the broker client to fetch the correct balance
	object, err := handler.accountObjectStore.Get(ctx, "TEST")
	if err != nil {
		return
	}
	broker := handler.brokerAdapterFactory.GetBrokerAdapter(ctx, object)
	balanceInfo, err := broker.GetBalanceInfo(ctx)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(balanceInfo)
	fatal.OnErrorUnlessDone(ctx, err)
}
