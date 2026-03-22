package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/stock-screener/internal/alpaca"
)

type Handler struct {
	alpacaClient alpaca.Client
}

type NewRouterInput struct {
	AlpacaClient alpaca.Client
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		alpacaClient: input.AlpacaClient,
	}
	router := mux.NewRouter().StrictSlash(true)
	stockScreenerV1Router := router.PathPrefix("/stock-screener/v1").Subrouter()
	// TODO: authorization
	stockScreenerV1Router.HandleFunc("/most-actives", handler.GetActiveStocks).Methods(http.MethodGet).Name("GetActiveStocks")
	stockScreenerV1Router.HandleFunc("/movers", handler.GetTopStockMovers).Methods(http.MethodGet).Name("GetTopStockMovers")
	return router
}
