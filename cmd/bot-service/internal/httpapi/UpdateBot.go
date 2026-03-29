package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
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
	case botstore.BotStatusRunning, botstore.BotStatusStopped:
	default:
		err = merry.New(`status must be "running" or "stopped"`).WithHTTPCode(http.StatusBadRequest)
		return
	}
	err = handler.botStore.UpdateBotStatus(ctx, botID, status)
	if err != nil {
		err = merryErrorByBotStoreError[err]
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
}
