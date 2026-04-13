package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/broker/alpaca"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httpx"
)

func (handler *Handler) GetStockBars(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	query := request.URL.Query()
	vars := mux.Vars(request)
	timeframe := query.Get("timeframe")
	if timeframe == "" {
		timeframe = "1Day"
	}
	limit := 365
	if v := query.Get("limit"); len(v) > 0 {
		limit, err = strconv.Atoi(v)
		if err != nil {
			return
		}
	}
	feed := query.Get("feed")
	if feed == "" {
		feed = "iex"
	}
	start := query.Get("start")
	end := query.Get("end")
	output, err := handler.alpacaClient.GetStockBars(ctx, alpaca.GetStockBarsInput{
		Symbol:    vars["symbol"],
		Timeframe: timeframe,
		Limit:     limit,
		Feed:      feed,
		Start:     start,
		End:       end,
	})
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(output)
	fatal.OnErrorUnlessDone(ctx, err)
}
