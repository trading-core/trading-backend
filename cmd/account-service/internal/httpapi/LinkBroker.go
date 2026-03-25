package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ansel1/merry"

	"github.com/kduong/trading-backend/cmd/account-service/internal/account"
	brokerType "github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/contextx"
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
	accountID := contextx.GetAccountID(ctx)

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
		err = merry.New("unsupported broker type").WithHTTPCode(http.StatusBadRequest).WithUserMessage("unsupported broker_type")
		return
	}

	current, err := handler.accountStore.Get(ctx, accountID)
	if err != nil {
		if !errors.Is(err, account.ErrNotFound) {
			return
		}
		current = &account.Account{ID: accountID}
		err = nil
	}

	current.BrokerType = input.BrokerType
	current.BrokerID = input.BrokerID
	err = handler.accountStore.Put(ctx, *current)
	if err != nil {
		return
	}

	httputil.SendResponseJSON(responseWriter, http.StatusOK, LinkBrokerOutput{
		AccountID:  accountID,
		BrokerType: current.BrokerType,
		BrokerID:   current.BrokerID,
		Linked:     true,
	})
}

func isSupportedBrokerType(rawBrokerType string) bool {
	switch rawBrokerType {
	case brokerType.TypeTastyTrade:
		return true
	default:
		return false
	}
}
