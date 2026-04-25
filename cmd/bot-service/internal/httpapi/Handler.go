package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/account-service/pkg/accountservice"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
	"github.com/kduong/trading-backend/cmd/bot-service/internal/symbolvalidator"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
)

const MaxActiveAllocationPercent = 80.0

type Handler struct {
	accountServiceClient   accountservice.Client
	symbolValidator        symbolvalidator.SymbolValidator
	botStoreCommandHandler botstore.CommandHandler
	botStoreQueryHandler   botstore.QueryHandler
	botEventLogFactory     eventsource.LogFactory
	botChannelFunc         func(botID string) string
}

type NewRouterInput struct {
	AuthMiddleware         *auth.Middleware
	AccountServiceClient   accountservice.Client
	SymbolValidator        symbolvalidator.SymbolValidator
	BotStoreCommandHandler botstore.CommandHandler
	BotStoreQueryHandler   botstore.QueryHandler
	BotEventLogFactory     eventsource.LogFactory
	BotChannelFunc         func(botID string) string
}

func NewRouter(input NewRouterInput) *mux.Router {
	symbolValidator := input.SymbolValidator
	if symbolValidator == nil {
		symbolValidator = symbolvalidator.NoopSymbolValidator{}
	}
	handler := &Handler{
		accountServiceClient:   input.AccountServiceClient,
		symbolValidator:        symbolValidator,
		botStoreCommandHandler: input.BotStoreCommandHandler,
		botStoreQueryHandler:   input.BotStoreQueryHandler,
		botEventLogFactory:     input.BotEventLogFactory,
		botChannelFunc:         input.BotChannelFunc,
	}
	router := mux.NewRouter().StrictSlash(true)
	botV1Router := router.PathPrefix("/bots/v1").Subrouter()
	botV1Router.Use(input.AuthMiddleware.Handle)
	botV1Router.HandleFunc("/bots", handler.CreateBot).Methods(http.MethodPost).Name("CreateBot")
	botV1Router.HandleFunc("/bots", handler.ListBots).Methods(http.MethodGet).Name("ListBots")
	botV1Router.HandleFunc("/bots/{bot_id}", handler.GetBot).Methods(http.MethodGet).Name("GetBot")
	botV1Router.HandleFunc("/bots/{bot_id}/stream", handler.StreamBotEvents).Methods(http.MethodGet).Name("StreamBotEvents")
	botV1Router.HandleFunc("/bots/{bot_id}", handler.UpdateBot).Methods(http.MethodPatch).Name("UpdateBot")
	botV1Router.HandleFunc("/bots/{bot_id}", handler.DeleteBot).Methods(http.MethodDelete).Name("DeleteBot")
	return router
}

func merrifyError(err error) error {
	switch {
	case errors.Is(err, botstore.ErrBotNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("bot not found")
	case errors.Is(err, botstore.ErrBotForbidden):
		return merry.Wrap(err).WithHTTPCode(http.StatusForbidden).WithUserMessage("forbidden")
	case errors.Is(err, accountservice.ErrAccountNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("account not found")
	case errors.Is(err, accountservice.ErrAccountForbidden):
		return merry.Wrap(err).WithHTTPCode(http.StatusForbidden).WithUserMessage("forbidden")
	case errors.Is(err, accountservice.ErrServerError):
		return merry.Wrap(err).WithHTTPCode(http.StatusInternalServerError).WithUserMessage("account service error")
	}
	return err
}

func ContextWithAccessTokenFromRequestHeader(ctx context.Context, request *http.Request) context.Context {
	authorization := request.Header.Get("Authorization")
	parts := strings.SplitN(authorization, " ", 2)
	fatal.Unless(len(parts) == 2, "invalid authorization header format")
	return contextx.WithAccessToken(ctx, parts[1])
}
