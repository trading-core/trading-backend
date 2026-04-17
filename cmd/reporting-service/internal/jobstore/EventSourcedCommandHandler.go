package jobstore

import (
	"context"

	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/logger"
)

var _ CommandHandler = (*EventSourcedCommandHandler)(nil)

type EventSourcedCommandHandler struct {
	log     eventsource.Log
	cursor  int64
	jobByID map[string]*Job
}

type NewEventSourcedCommandHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedCommandHandler(input NewEventSourcedCommandHandlerInput) *EventSourcedCommandHandler {
	return &EventSourcedCommandHandler{
		log:     input.Log,
		jobByID: make(map[string]*Job),
	}
}

func (store *EventSourcedCommandHandler) CreateJob(ctx context.Context, job *Job) error {
	store.catchUp(ctx)
	if _, exists := store.jobByID[job.ID]; exists {
		logger.Fatalf("job with ID %s already exists", job.ID)
	}
	fatal.Unless(job.Status == JobStatusPending, "new job must have status pending")
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeJobEnqueued),
		JobEnqueuedEvent: &JobEnqueuedEvent{
			JobID:      job.ID,
			UserID:     job.UserID,
			Name:       job.Name,
			Kind:       job.Kind,
			Parameters: job.Parameters,
			CreatedAt:  job.CreatedAt,
		},
	})
	_, err := store.log.Append(payload)
	return err
}

func (store *EventSourcedCommandHandler) UpdateJobStatus(ctx context.Context, input UpdateJobStatusInput) (err error) {
	store.catchUp(ctx)
	if _, ok := store.jobByID[input.JobID]; !ok {
		return ErrJobNotFound
	}
	var payload []byte
	switch input.Status {
	case JobStatusRunning:
		payload = fatal.UnlessMarshal(EventFrame{
			EventBase:       eventsource.NewEventBase(EventTypeJobStarted),
			JobStartedEvent: &JobStartedEvent{JobID: input.JobID, UpdatedAt: input.UpdatedAt},
		})
	case JobStatusCompleted:
		payload = fatal.UnlessMarshal(EventFrame{
			EventBase:         eventsource.NewEventBase(EventTypeJobCompleted),
			JobCompletedEvent: &JobCompletedEvent{JobID: input.JobID, DownloadURL: input.DownloadURL, UpdatedAt: input.UpdatedAt},
		})
	case JobStatusFailed:
		payload = fatal.UnlessMarshal(EventFrame{
			EventBase:      eventsource.NewEventBase(EventTypeJobFailed),
			JobFailedEvent: &JobFailedEvent{JobID: input.JobID, FailReason: input.FailReason, UpdatedAt: input.UpdatedAt},
		})
	case JobStatusPending:
		payload = fatal.UnlessMarshal(EventFrame{
			EventBase:       eventsource.NewEventBase(EventTypeJobRetried),
			JobRetriedEvent: &JobRetriedEvent{JobID: input.JobID, RetryCount: input.RetryCount, UpdatedAt: input.UpdatedAt},
		})
	default:
		logger.Fatalf("unhandled job status %s", input.Status)
	}
	_, err = store.log.Append(payload)
	fatal.OnError(err)
	return
}

func (store *EventSourcedCommandHandler) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedCommandHandler) apply(ctx context.Context, event *eventsource.Event) error {
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

func (store *EventSourcedCommandHandler) applyRetried(event *JobRetriedEvent) error {
	job, ok := store.jobByID[event.JobID]
	if !ok {
		return nil
	}
	job.RetryCount = event.RetryCount
	job.Status = JobStatusPending
	job.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedCommandHandler) applyEnqueued(event *JobEnqueuedEvent) error {
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

func (store *EventSourcedCommandHandler) applyStarted(event *JobStartedEvent) error {
	job, ok := store.jobByID[event.JobID]
	if !ok {
		return nil
	}
	job.Status = JobStatusRunning
	job.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedCommandHandler) applyCompleted(event *JobCompletedEvent) error {
	job, ok := store.jobByID[event.JobID]
	if !ok {
		return nil
	}
	job.Status = JobStatusCompleted
	job.DownloadURL = event.DownloadURL
	job.UpdatedAt = event.UpdatedAt
	return nil
}

func (store *EventSourcedCommandHandler) applyFailed(event *JobFailedEvent) error {
	job, ok := store.jobByID[event.JobID]
	if !ok {
		return nil
	}
	job.Status = JobStatusFailed
	job.FailReason = event.FailReason
	job.UpdatedAt = event.UpdatedAt
	return nil
}
