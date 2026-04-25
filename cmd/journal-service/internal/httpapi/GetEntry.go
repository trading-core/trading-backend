package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/httpx"
)

func (handler *Handler) GetEntry(responseWriter http.ResponseWriter, request *http.Request) {
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
	entry, err := handler.entryQueryHandler.Get(ctx, date)
	if err != nil {
		err = merrifyError(err)
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(entry)
}
