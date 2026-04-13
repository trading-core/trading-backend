package httpapi

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/symbolvalidator"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/httpx"
	"github.com/kduong/trading-backend/internal/tradingstrategy"
	uuid "github.com/satori/go.uuid"
)

var symbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,14}$`)

type CreateBotInput struct {
	AccountID         string                      `json:"account_id"`
	Symbol            string                      `json:"symbol"`
	AllocationPercent float64                     `json:"allocation_percent"`
	TradingParameters *tradingstrategy.Parameters `json:"trading_params,omitempty"`
}

func (input *CreateBotInput) Validate() (err error) {
	if input.AccountID == "" || input.Symbol == "" {
		err = merry.New("account_id and symbol are required").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if !symbolPattern.MatchString(input.Symbol) {
		err = merry.New("symbol must be 1-15 chars using A-Z, 0-9, '.', or '-'").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if math.IsNaN(input.AllocationPercent) || math.IsInf(input.AllocationPercent, 0) {
		err = merry.New("allocation_percent must be a valid number").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if input.AllocationPercent <= 0 || input.AllocationPercent > MaxActiveAllocationPercent {
		err = merry.New("allocation_percent must be greater than 0 and less than or equal to 80").WithHTTPCode(http.StatusBadRequest)
		return
	}
	return
}

func (handler *Handler) CreateBot(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	var input CreateBotInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest)
		return
	}
	err = input.Validate()
	if err != nil {
		return
	}
	ctx = ContextWithAccessTokenFromRequestHeader(ctx, request)
	account, err := handler.accountServiceClient.GetAccount(ctx, input.AccountID)
	if err != nil {
		err = merrifyError[err]
		return
	}
	if !account.BrokerLinked {
		err = merry.New("account is not linked to a broker").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if account.Broker == nil {
		err = merry.New("account broker details are missing").WithHTTPCode(http.StatusBadRequest)
		return
	}
	err = handler.symbolValidator.Validate(ctx, account.Broker.Type, input.Symbol)
	if err != nil {
		switch {
		case errors.Is(err, symbolvalidator.ErrSymbolNotTradableForBroker):
			err = merry.New("symbol is not tradable for this account broker").WithHTTPCode(http.StatusBadRequest)
		case errors.Is(err, symbolvalidator.ErrUnsupportedBrokerForSymbolValidation):
			err = merry.New("account broker is not supported for symbol validation").WithHTTPCode(http.StatusBadRequest)
		default:
			err = merry.Wrap(err).WithHTTPCode(http.StatusBadGateway)
		}
		return
	}
	bot := &botstore.Bot{
		ID:                uuid.NewV4().String(),
		UserID:            userID,
		AccountID:         account.ID,
		BrokerAccountID:   account.Broker.ID,
		BrokerType:        account.Broker.Type,
		Symbol:            input.Symbol,
		AllocationPercent: input.AllocationPercent,
		TradingParameters: input.TradingParameters,
		Status:            botstore.BotStatusStopped,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	err = handler.botStoreCommandHandler.Create(ctx, bot)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(bot)
}
