package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/broker/alpaca"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/httputil"
)

func (handler *Handler) GetStockSnapshot(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	symbol := vars["symbol"]
	output, err := handler.alpacaClient.GetStockSnapshot(ctx, alpaca.GetStockSnapshotInput{
		Symbol: symbol,
	})
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(responseWriter).Encode(output)
	fatal.OnErrorUnlessDone(ctx, err)
}
