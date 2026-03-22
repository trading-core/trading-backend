package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/kduong/trading-backend/cmd/stock-screener/internal/alpaca"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) GetStockNews(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	query := request.URL.Query()
	var symbols []string
	if v := query.Get("symbols"); len(v) > 0 {
		symbols = strings.Split(v, ",")
	}
	limit := 10
	if v := query.Get("limit"); len(v) > 0 {
		limit, err = strconv.Atoi(v)
		if err != nil {
			return
		}
	}
	nextPageToken := query.Get("next_page_token")
	output, err := handler.alpacaClient.GetStockNews(ctx, alpaca.GetStockNewsInput{
		NextPageToken: nextPageToken,
		Symbols:       symbols,
		Limit:         limit,
	})
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(output)
	fatal.OnErrorUnlessDone(ctx, err)
}
