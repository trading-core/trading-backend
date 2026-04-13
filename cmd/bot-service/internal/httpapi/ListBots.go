package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/kduong/trading-backend/internal/httpx"
)

func (handler *Handler) ListBots(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	bots, err := handler.botStoreQueryHandler.List(ctx)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(bots)
}
