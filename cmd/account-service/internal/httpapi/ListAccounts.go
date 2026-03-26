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
}
