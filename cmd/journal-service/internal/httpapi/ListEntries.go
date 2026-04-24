package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/journal-service/internal/entrystore"
	"github.com/kduong/trading-backend/internal/httpx"
)

const defaultPageSize = 31
const maxPageSize = 366

func (handler *Handler) ListEntries(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()

	from := request.URL.Query().Get("from")
	to := request.URL.Query().Get("to")
	if from != "" {
		if _, parseErr := time.Parse(dateLayout, from); parseErr != nil {
			err = merry.New("from must be YYYY-MM-DD").WithHTTPCode(http.StatusBadRequest)
			return
		}
	}
	if to != "" {
		if _, parseErr := time.Parse(dateLayout, to); parseErr != nil {
			err = merry.New("to must be YYYY-MM-DD").WithHTTPCode(http.StatusBadRequest)
			return
		}
	}

	page, err := parseQueryInt(request, "page", 0)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest)
		return
	}
	pageSize, err := parseQueryInt(request, "page_size", defaultPageSize)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest)
		return
	}
	if pageSize < 1 || pageSize > maxPageSize {
		err = merry.New("page_size must be between 1 and 366").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if page < 0 {
		err = merry.New("page must be >= 0").WithHTTPCode(http.StatusBadRequest)
		return
	}

	result, err := handler.entryQueryHandler.List(ctx, entrystore.ListInput{
		From:     from,
		To:       to,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(result)
}

func parseQueryInt(request *http.Request, key string, defaultValue int) (int, error) {
	raw := request.URL.Query().Get(key)
	if raw == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(raw)
}
