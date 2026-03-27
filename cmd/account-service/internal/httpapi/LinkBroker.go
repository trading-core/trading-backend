package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type LinkBrokerOutput struct {
	AccountID     string         `json:"account_id"`
	BrokerAccount broker.Account `json:"broker_account"`
}

func (handler *Handler) LinkBroker(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	accountID := vars["account_id"]
	var input broker.Account
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	err = handler.accountStore.LinkBrokerAccount(ctx, account.LinkBrokerAccountInput{
		AccountID:     accountID,
		BrokerAccount: &input,
	})
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(LinkBrokerOutput{
		AccountID:     accountID,
		BrokerAccount: input,
	})
	fatal.OnErrorUnlessDone(ctx, err)
}
