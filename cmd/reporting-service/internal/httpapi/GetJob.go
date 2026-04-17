package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/httpx"
)

func (handler *Handler) GetJob(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	jobID := vars["job_id"]
	job, err := handler.jobQueryHandler.Get(ctx, jobID)
	if err != nil {
		err = merrifyError[err]
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(job)
}
