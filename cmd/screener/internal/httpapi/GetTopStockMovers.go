package httpapi

import (
	"errors"
	"net/http"

	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) GetTopStockMovers(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	err = errors.New("test")
}
