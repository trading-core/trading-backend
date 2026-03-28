package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker/alpaca"
)

type Handler struct {
	alpacaClient alpaca.Client
}

type NewRouterInput struct {
	AlpacaClient   alpaca.Client
	AuthMiddleware *auth.Middleware
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		alpacaClient: input.AlpacaClient,
	}
	router := mux.NewRouter().StrictSlash(true)
	stockScreenerV1Router := router.PathPrefix("/stock-screener/v1").Subrouter()
	stockScreenerV1Router.Use(input.AuthMiddleware.Handle)
	stockScreenerV1Router.HandleFunc("/most-actives", handler.GetActiveStocks).Methods(http.MethodGet).Name("GetActiveStocks")
	stockScreenerV1Router.HandleFunc("/movers", handler.GetTopStockMovers).Methods(http.MethodGet).Name("GetTopStockMovers")
	stockScreenerV1Router.HandleFunc("/news", handler.GetStockNews).Methods(http.MethodGet).Name("GetStockNews")
	stockScreenerV1Router.HandleFunc("/stocks/{symbol}/snapshot", handler.GetStockSnapshot).Methods(http.MethodGet).Name("GetStockSnapshot")
	return router
}
