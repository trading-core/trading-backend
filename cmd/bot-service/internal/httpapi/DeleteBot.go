package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/httpx"
)

func (handler *Handler) DeleteBot(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	botID := vars["bot_id"]
	err = handler.botStoreCommandHandler.Delete(ctx, botID)
	if err != nil {
		err = merrifyError[err]
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
}
