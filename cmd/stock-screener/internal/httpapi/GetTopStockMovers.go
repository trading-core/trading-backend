package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/kduong/trading-backend/cmd/stock-screener/internal/alpaca"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) GetTopStockMovers(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	query := request.URL.Query()
	limit := 10
	if v := query.Get("limit"); len(v) > 0 {
		limit, err = strconv.Atoi(v)
		if err != nil {
			return
		}
	}
	output, err := handler.alpacaClient.GetTopStockMovers(ctx, alpaca.GetTopStockMoversInput{
		Limit: limit,
	})
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(output)
	fatal.OnErrorUnlessDone(ctx, err)
}
