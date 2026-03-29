package httpapi

import (
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/account-service/pkg/accountservice"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/internal/auth"
)

type Handler struct {
	accountServiceClient accountservice.Client
	botStore             botstore.Store
}

type NewRouterInput struct {
	AuthMiddleware       *auth.Middleware
	AccountServiceClient accountservice.Client
	BotStore             botstore.Store
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		accountServiceClient: input.AccountServiceClient,
		botStore:             input.BotStore,
	}
	router := mux.NewRouter().StrictSlash(true)
	botV1Router := router.PathPrefix("/bots/v1").Subrouter()
	botV1Router.Use(input.AuthMiddleware.Handle)
	botV1Router.HandleFunc("/bots", handler.CreateBot).Methods(http.MethodPost).Name("CreateBot")
	botV1Router.HandleFunc("/bots", handler.ListBots).Methods(http.MethodGet).Name("ListBots")
	botV1Router.HandleFunc("/bots/{bot_id}", handler.GetBot).Methods(http.MethodGet).Name("GetBot")
	botV1Router.HandleFunc("/bots/{bot_id}", handler.UpdateBot).Methods(http.MethodPatch).Name("UpdateBot")
	botV1Router.HandleFunc("/bots/{bot_id}", handler.DeleteBot).Methods(http.MethodDelete).Name("DeleteBot")
	return router
}

var merryErrorByBotStoreError = map[error]error{
	botstore.ErrBotNotFound:  merry.New("bot not found").WithHTTPCode(http.StatusNotFound),
	botstore.ErrBotForbidden: merry.New("forbidden").WithHTTPCode(http.StatusForbidden),
}
