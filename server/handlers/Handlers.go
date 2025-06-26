package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Handlers struct {
}

func (handlers *Handlers) AddRoutes(router *mux.Router) {
	tradingRouter := router.PathPrefix("/trading/v1").Subrouter()
	tradingRouter.HandleFunc("/order", handlers.CreateOrder).Methods(http.MethodPost).Name("CreateOrder")
}

func (handlers *Handlers) CreateOrder(responseWriter http.ResponseWriter, request *http.Request) {
}
