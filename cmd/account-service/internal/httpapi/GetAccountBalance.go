package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/account-service/internal/accountstore"
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
	account, err := handler.accountStoreQueryHandler.Get(ctx, accountstore.GetInput{
		AccountID: accountID,
	})
	if err != nil {
		err = merryErrorByAccountStoreError[err]
		return
	}
	if !account.BrokerLinked {
		err = merry.New("account is not linked to a broker").WithHTTPCode(http.StatusBadRequest)
		return
	}
	accountClient := handler.brokerAccountClientFactory.Get(ctx, account.BrokerAccount)
	output, err := accountClient.GetBalance(ctx)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(output)
	fatal.OnErrorUnlessDone(ctx, err)
}
