package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/jobstore"
	"github.com/kduong/trading-backend/internal/httpx"
)

const defaultPageSize = 10
const maxPageSize = 100

func (handler *Handler) ListJobs(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()

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
		err = merry.New("page_size must be between 1 and 100").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if page < 0 {
		err = merry.New("page must be >= 0").WithHTTPCode(http.StatusBadRequest)
		return
	}

	result, err := handler.jobQueryHandler.List(ctx, jobstore.ListInput{
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
