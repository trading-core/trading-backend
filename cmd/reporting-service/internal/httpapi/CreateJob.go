package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/jobstore"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/httpx"
	uuid "github.com/satori/go.uuid"
)

type EnqueueJobInput struct {
	Kind       string            `json:"kind"`
	Name       string            `json:"name,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

func (input *EnqueueJobInput) Validate() error {
	if input.Kind == "" {
		return merry.New("kind is required").WithHTTPCode(http.StatusBadRequest)
	}
	return nil
}

func (handler *Handler) CreateJob(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	userID := contextx.GetUserID(ctx)
	var input EnqueueJobInput
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
	job := &jobstore.Job{
		ID:         uuid.NewV4().String(),
		UserID:     userID,
		Name:       input.Name,
		Kind:       input.Kind,
		Parameters: input.Parameters,
		Status:     jobstore.JobStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err = handler.jobCommandHandler.CreateJob(ctx, job)
	if err != nil {
		return
	}
	// Notify the actor non-blocking; the actor's channel is buffered and the
	// recovery pass on restart handles any jobs that don't make it through.
	handler.enqueueJob(job)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusAccepted)
	json.NewEncoder(responseWriter).Encode(job)
}
