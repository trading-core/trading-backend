package httpapi

import (
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/journal-service/internal/entrystore"
	"github.com/kduong/trading-backend/internal/auth"
)

type Handler struct {
	entryCommandHandler entrystore.CommandHandler
	entryQueryHandler   entrystore.QueryHandler
}

type NewRouterInput struct {
	AuthMiddleware      *auth.Middleware
	EntryCommandHandler entrystore.CommandHandler
	EntryQueryHandler   entrystore.QueryHandler
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		entryCommandHandler: input.EntryCommandHandler,
		entryQueryHandler:   input.EntryQueryHandler,
	}
	router := mux.NewRouter().StrictSlash(true)
	journalV1Router := router.PathPrefix("/journal/v1").Subrouter()
	journalV1Router.Use(input.AuthMiddleware.Handle)
	journalV1Router.HandleFunc("/entries", handler.ListEntries).Methods(http.MethodGet).Name("ListEntries")
	journalV1Router.HandleFunc("/entries/{date}", handler.GetEntry).Methods(http.MethodGet).Name("GetEntry")
	journalV1Router.HandleFunc("/entries/{date}", handler.UpsertEntry).Methods(http.MethodPut).Name("UpsertEntry")
	journalV1Router.HandleFunc("/entries/{date}", handler.DeleteEntry).Methods(http.MethodDelete).Name("DeleteEntry")
	return router
}

func merrifyError(err error) error {
	switch {
	case errors.Is(err, entrystore.ErrEntryNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("entry not found")
	case errors.Is(err, entrystore.ErrEntryForbidden):
		return merry.Wrap(err).WithHTTPCode(http.StatusForbidden).WithUserMessage("forbidden")
	}
	return err
}
