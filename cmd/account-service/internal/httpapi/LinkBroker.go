package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type LinkBrokerInput struct {
	BrokerType string `json:"broker_type"`
	BrokerID   string `json:"broker_id"`
}

type LinkBrokerOutput struct {
	AccountID  string `json:"account_id"`
	BrokerType string `json:"broker_type"`
	BrokerID   string `json:"broker_id"`
	Linked     bool   `json:"linked"`
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
	var input LinkBrokerInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
		return
	}
	input.BrokerType = strings.TrimSpace(strings.ToLower(input.BrokerType))
	input.BrokerID = strings.TrimSpace(input.BrokerID)
	if len(input.BrokerType) == 0 || len(input.BrokerID) == 0 {
		err = merry.New("broker_type and broker_id are required").WithHTTPCode(http.StatusBadRequest).WithUserMessage("broker_type and broker_id are required")
		return
	}
	if !isSupportedBrokerType(input.BrokerType) {
		err = merry.New("unsupported broker_type").WithHTTPCode(http.StatusBadRequest)
		return
	}
	// TODO: validate broker_id format based on broker_type
	err = handler.accountStore.LinkBrokerAccount(ctx, account.LinkBrokerAccountInput{
		AccountID: accountID,
		BrokerAccount: &broker.Account{
			Type: input.BrokerType,
			ID:   input.BrokerID,
		},
	})
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(LinkBrokerOutput{
		AccountID:  accountID,
		BrokerType: input.BrokerType,
		BrokerID:   input.BrokerID,
		Linked:     true,
	})
	fatal.OnErrorUnlessDone(ctx, err)
}

func isSupportedBrokerType(brokerType string) bool {
	switch brokerType {
	case "tastytrade":
		return true
	default:
		return false
	}
}
