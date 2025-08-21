package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong/tradingbot/internal/bybit"
)

type HTTPAPI struct {
	Client bybit.Client
}

func (httpAPI *HTTPAPI) AddRoutes(router *mux.Router) {
	tradingV1Router := router.PathPrefix("/trading/v1").Subrouter()
	tradingV1Router.HandleFunc("/order", httpAPI.CreateOrder).Methods(http.MethodPost).Name("CreateOrder")
	tradingV1Router.HandleFunc("/balance", httpAPI.GetBalance).Methods(http.MethodGet).Name("GetBalance")
	//
}
