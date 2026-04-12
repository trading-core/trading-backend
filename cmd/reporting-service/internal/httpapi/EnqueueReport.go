package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportstore"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/httputil"
	uuid "github.com/satori/go.uuid"
)

type EnqueueReportInput struct {
	Kind       string            `json:"kind"`
	Name       string            `json:"name,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

func (input *EnqueueReportInput) Validate() error {
	if input.Kind == "" {
		return merry.New("kind is required").WithHTTPCode(http.StatusBadRequest)
	}
	return nil
}

func (handler *Handler) EnqueueReport(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	var input EnqueueReportInput
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest)
		return
	}
	err = input.Validate()
	if err != nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	report := &reportstore.Report{
		ID:         uuid.NewV4().String(),
		UserID:     userID,
		Name:       input.Name,
		Kind:       input.Kind,
		Parameters: input.Parameters,
		Status:     reportstore.ReportStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err = handler.reportCommandHandler.Enqueue(ctx, report)
	if err != nil {
		return
	}
	// Notify the worker non-blocking; the channel is buffered and the recovery
	// worker handles any jobs that don't make it through on a restart.
	select {
	case handler.jobs <- report.ID:
	default:
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusAccepted)
	json.NewEncoder(responseWriter).Encode(report)
}
