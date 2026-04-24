package httpapi

import (
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/journal-service/internal/entrystore"
	"github.com/kduong/trading-backend/internal/httpx"
)

func (handler *Handler) DeleteEntry(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	date := vars["date"]
	if _, parseErr := time.Parse(dateLayout, date); parseErr != nil {
		err = merry.New("date must be YYYY-MM-DD").WithHTTPCode(http.StatusBadRequest)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = handler.entryCommandHandler.DeleteEntry(ctx, entrystore.DeleteEntryInput{
		Date:      date,
		UpdatedAt: now,
	})
	if err != nil {
		err = merrifyError[err]
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
