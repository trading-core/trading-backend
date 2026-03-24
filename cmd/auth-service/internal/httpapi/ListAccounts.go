package httpapi

import (
	"net/http"

	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) ListAccounts(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	objects, err := handler.accountStore.List(ctx)
	if err != nil {
		return
	}
	httputil.SendResponseJSON(responseWriter, http.StatusOK, objects)
}
