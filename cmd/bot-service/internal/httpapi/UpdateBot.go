package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

type UpdateBotInput struct {
	Status string `json:"status"`
}

func (handler *Handler) UpdateBot(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	botID := vars["bot_id"]
	var body UpdateBotInput
	err = json.NewDecoder(request.Body).Decode(&body)
	if err != nil {
		return
	}
	status := botstore.BotStatus(body.Status)
	switch status {
	case botstore.BotStatusRunning:
		err = handler.ensureAllocationPolicy(ctx, request, botID)
		if err != nil {
			return
		}
	case botstore.BotStatusStopped:
	default:
		err = merry.New(`status must be "running" or "stopped"`).WithHTTPCode(http.StatusBadRequest)
		return
	}
	err = handler.botStore.UpdateBotStatus(ctx, botID, status)
	if err != nil {
		err = merrifyError[err]
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
}

func (handler *Handler) ensureAllocationPolicy(ctx context.Context, request *http.Request, botID string) (err error) {
	bot, err := handler.botStore.Get(ctx, botID)
	if err != nil {
		err = merrifyError[err]
		return
	}
	ctx = ContextWithAccessTokenFromRequestHeader(ctx, request)
	balance, err := handler.accountServiceClient.GetAccountBalance(ctx, bot.AccountID)
	if err != nil {
		err = merrifyError[err]
		return
	}
	if balance.CashBalance <= 0 {
		err = merry.New("account has no available cash balance").WithHTTPCode(http.StatusBadRequest)
		return
	}
	bots, err := handler.botStore.List(ctx)
	fatal.OnError(err)
	activeAllocationPercent := 0.0
	for _, botItem := range bots {
		if botItem.ID == botID {
			continue
		}
		if botItem.AccountID != bot.AccountID {
			continue
		}
		if botItem.Status != botstore.BotStatusRunning {
			continue
		}
		activeAllocationPercent += bot.AllocationPercent
	}
	if activeAllocationPercent+bot.AllocationPercent > MaxActiveAllocationPercent {
		err = merry.New("active bot allocation exceeds 80% for this account").WithHTTPCode(http.StatusBadRequest)
		return
	}
	return
}
