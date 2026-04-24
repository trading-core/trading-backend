package jobstore

import (
	"context"
	"sort"

	"github.com/kduong/trading-backend/internal/authz"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ QueryHandler = (*EventSourcedQueryHandler)(nil)

type EventSourcedQueryHandler struct {
	log      eventsource.Log
	cursor   int64
	jobByID  map[string]*Job
}

type NewEventSourcedQueryHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedQueryHandler(input NewEventSourcedQueryHandlerInput) *EventSourcedQueryHandler {
	return &EventSourcedQueryHandler{
		log:     input.Log,
		jobByID: make(map[string]*Job),
	}
}

func (store *EventSourcedQueryHandler) Get(ctx context.Context, jobID string) (job *Job, err error) {
	store.catchUp(ctx)
	job, ok := store.jobByID[jobID]
	if !ok {
		err = ErrJobNotFound
		return
	}
	if ownershipErr := authz.RequireOwner(ctx, job.UserID); ownershipErr != nil {
		err = ErrJobForbidden
		return
	}
	return
}

func (store *EventSourcedQueryHandler) GetSystem(ctx context.Context, jobID string) (job *Job, err error) {
	store.catchUp(ctx)
	job, ok := store.jobByID[jobID]
	if !ok {
		err = ErrJobNotFound
		return
	}
	return
}

func (store *EventSourcedQueryHandler) List(ctx context.Context, input ListInput) (*ListResult, error) {
	store.catchUp(ctx)
	userID := contextx.GetUserID(ctx)
	all := make([]*Job, 0)
	for _, job := range store.jobByID {
		if job.UserID != userID {
			continue
		}
		all = append(all, job)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt > all[j].CreatedAt
	})
	totalCount := len(all)
	totalPages := totalCount / input.PageSize
	if totalCount%input.PageSize != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}
	start := input.Page * input.PageSize
	if start > totalCount {
		start = totalCount
	}
	end := start + input.PageSize
	if end > totalCount {
		end = totalCount
	}
	return &ListResult{
		Jobs:       all[start:end],
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func (store *EventSourcedQueryHandler) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedQueryHandler) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeJobEnqueued:
		return store.applyEnqueued(frame.JobEnqueuedEvent)
	case EventTypeJobStarted:
		return store.applyStarted(frame.JobStartedEvent)
	case EventTypeJobCompleted:
		return store.applyCompleted(frame.JobCompletedEvent)
	case EventTypeJobFailed:
		return store.applyFailed(frame.JobFailedEvent)
	case EventTypeJobRetried:
		return store.applyRetried(frame.JobRetriedEvent)
	}
	return nil
}

func (store *EventSourcedQueryHandler) applyRetried(event *JobRetriedEvent) error {
	job, ok := store.jobByID[event.JobID]
	if !ok {
		return nil
	}
	job.RetryCount = event.RetryCount
	job.Status = JobStatusPending
	job.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedQueryHandler) applyEnqueued(event *JobEnqueuedEvent) error {
	store.jobByID[event.JobID] = &Job{
		ID:         event.JobID,
		UserID:     event.UserID,
		Name:       event.Name,
		Kind:       event.Kind,
		Parameters: event.Parameters,
		Status:     JobStatusPending,
		CreatedAt:  event.CreatedAt,
		UpdatedAt:  event.CreatedAt,
	}
	return nil
}

func (store *EventSourcedQueryHandler) applyStarted(event *JobStartedEvent) error {
	job, ok := store.jobByID[event.JobID]
	if !ok {
		return nil
	}
	job.Status = JobStatusRunning
	job.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedQueryHandler) applyCompleted(event *JobCompletedEvent) error {
	job, ok := store.jobByID[event.JobID]
	if !ok {
		return nil
	}
	job.Status = JobStatusCompleted
	job.DownloadURL = event.DownloadURL
	job.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedQueryHandler) applyFailed(event *JobFailedEvent) error {
	job, ok := store.jobByID[event.JobID]
	if !ok {
		return nil
	}
	job.Status = JobStatusFailed
	job.FailReason = event.FailReason
	job.UpdatedAt = event.UpdatedAt
	return nil
}
