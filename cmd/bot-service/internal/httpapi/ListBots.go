package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) ListBots(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	bots, err := handler.botStore.List(ctx)
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(bots)
}
