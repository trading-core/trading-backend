package httpapi

import (
	"context"
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
	if err = handler.ValidateBrokerAccount(ctx, &input); err != nil {
		return
	}
	err = handler.accountStore.LinkBrokerAccount(ctx, account.LinkBrokerAccountInput{
		AccountID:     accountID,
		BrokerAccount: &input,
	})
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(LinkBrokerOutput{
		AccountID:     accountID,
		BrokerAccount: input,
	})
	fatal.OnErrorUnlessDone(ctx, err)
}

var validBrokerAccountTypes = map[broker.AccountType]struct{}{
	broker.AccountTypeTastyTrade: {},
}

func (handler *Handler) ValidateBrokerAccount(ctx context.Context, brokerAccount *broker.Account) error {
	if _, isValidBrokerAccountType := validBrokerAccountTypes[brokerAccount.Type]; !isValidBrokerAccountType {
		return merry.Errorf("unsupported broker account type: %s", brokerAccount.Type).WithHTTPCode(http.StatusBadRequest)
	}
	broker := handler.brokerClientFactory.GetClient(ctx, brokerAccount)
	if _, err := broker.GetBalanceInfo(ctx); err != nil {
		return merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("failed to validate broker account")
	}
	return nil

}
