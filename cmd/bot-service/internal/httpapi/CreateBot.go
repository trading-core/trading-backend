package httpapi

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/httputil"
	uuid "github.com/satori/go.uuid"
)

type CreateBotInput struct {
	AccountID         string  `json:"account_id"`
	Symbol            string  `json:"symbol"`
	StrategyTradeType string  `json:"strategy_trade_type"`
	AllocationPercent float64 `json:"allocation_percent"`
}

func (input *CreateBotInput) Validate() (err error) {
	if input.AccountID == "" || input.Symbol == "" || input.StrategyTradeType == "" {
		err = merry.New("account_id, symbol, and strategy_trade_type are required").WithHTTPCode(http.StatusBadRequest)
		return
	}
	strategy := tradingstrategy.New(input.StrategyTradeType)
	err = tradingstrategy.Validate(strategy)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest)
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
			httputil.SendErrorResponse(responseWriter, err)
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
	bot := &botstore.Bot{
		ID:                uuid.NewV4().String(),
		UserID:            userID,
		AccountID:         account.ID,
		BrokerAccountID:   account.Broker.ID,
		BrokerType:        account.Broker.Type,
		Symbol:            input.Symbol,
		StrategyTradeType: input.StrategyTradeType,
		AllocationPercent: input.AllocationPercent,
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
