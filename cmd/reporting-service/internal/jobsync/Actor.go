package jobsync

import (
	"context"
	"fmt"
	"time"

	"github.com/kduong/trading-backend/cmd/reporting-service/internal/jobstore"
	"github.com/kduong/trading-backend/cmd/storage-service/pkg/storageservice"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/kduong/trading-backend/internal/logger"
)

const MaxRetries = 3

type Actor struct {
	jobStoreCommandHandler jobstore.CommandHandler
	storageClient          storageservice.Client
	serviceTokenMinter     *auth.ServiceTokenMinter
	outputsDirectory       string
	jobs                   chan *jobstore.Job
	log                    eventsource.Log
	incompleteJobsByID     map[string]*jobstore.Job
}

type NewActorInput struct {
	CommandHandler     jobstore.CommandHandler
	StorageClient      storageservice.Client
	ServiceTokenMinter *auth.ServiceTokenMinter
	OutputsDirectory   string
	Log                eventsource.Log
}

func NewActor(input NewActorInput) *Actor {
	return &Actor{
		jobStoreCommandHandler: input.CommandHandler,
		storageClient:          input.StorageClient,
		serviceTokenMinter:     input.ServiceTokenMinter,
		outputsDirectory:       input.OutputsDirectory,
		jobs:                   make(chan *jobstore.Job, 64),
		log:                    input.Log,
		incompleteJobsByID:     make(map[string]*jobstore.Job),
	}
}

// Notify enqueues a job for processing. It is safe to call from any goroutine.
// If the internal buffer is full the notification is dropped; the recovery pass
// on the next restart will pick it up.
func (actor *Actor) Notify(job *jobstore.Job) {
	select {
	case actor.jobs <- job:
	default:
	}
}

func (actor *Actor) CatchUp(ctx context.Context) {
	_, err := subscription.CatchUp(ctx, subscription.Input{
		Log:   actor.log,
		Apply: actor.apply,
	})
	fatal.OnError(err)
}

func (actor *Actor) apply(ctx context.Context, event *eventsource.Event) error {
	var frame jobstore.EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case jobstore.EventTypeJobEnqueued:
		return actor.applyEnqueued(frame.JobEnqueuedEvent)
	case jobstore.EventTypeJobStarted:
		return actor.applyStarted(frame.JobStartedEvent)
	case jobstore.EventTypeJobRetried:
		return actor.applyRetried(frame.JobRetriedEvent)
	}
	return nil
}

func (actor *Actor) applyEnqueued(event *jobstore.JobEnqueuedEvent) error {
	actor.incompleteJobsByID[event.JobID] = &jobstore.Job{
		ID:         event.JobID,
		UserID:     event.UserID,
		Name:       event.Name,
		Kind:       event.Kind,
		Parameters: event.Parameters,
		Status:     jobstore.JobStatusPending,
		CreatedAt:  event.CreatedAt,
		UpdatedAt:  event.CreatedAt,
	}
	return nil
}

func (actor *Actor) applyStarted(event *jobstore.JobStartedEvent) error {
	job, ok := actor.incompleteJobsByID[event.JobID]
	if !ok {
		return nil
	}
	job.Status = jobstore.JobStatusRunning
	job.UpdatedAt = event.UpdatedAt
	return nil
}

func (actor *Actor) applyRetried(event *jobstore.JobRetriedEvent) error {
	job, ok := actor.incompleteJobsByID[event.JobID]
	if !ok {
		return nil
	}
	job.RetryCount = event.RetryCount
	job.Status = jobstore.JobStatusPending
	job.UpdatedAt = event.UpdatedAt
	return nil
}

func (actor *Actor) CompleteCatchup(ctx context.Context) {
	recovered := 0
	deadLettered := 0
	for _, job := range actor.incompleteJobsByID {
		now := time.Now().UTC().Format(time.RFC3339)
		nextRetry := job.RetryCount + 1
		if nextRetry > MaxRetries {
			err := actor.jobStoreCommandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{
				JobID:      job.ID,
				Status:     jobstore.JobStatusFailed,
				FailReason: fmt.Sprintf("dead letter: job failed to complete after %d attempts", MaxRetries),
				UpdatedAt:  now,
			})
			if err != nil {
				logger.Warnpf("reportsync: recover: could not dead-letter job %s: %v", job.ID, err)
			} else {
				deadLettered++
			}
			continue
		}
		err := actor.jobStoreCommandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{
			JobID:      job.ID,
			Status:     jobstore.JobStatusPending,
			RetryCount: job.RetryCount + 1,
			UpdatedAt:  now,
		})
		if err != nil {
			logger.Warnpf("reportsync: recover: could not increment retry for job %s: %v", job.ID, err)
			continue
		}
		actor.jobs <- job
		recovered++
	}
	actor.incompleteJobsByID = nil
	if recovered > 0 {
		logger.Warnpf("reportsync: recover: requeued %d interrupted job(s)", recovered)
	}
	if deadLettered > 0 {
		logger.Warnpf("reportsync: recover: dead-lettered %d job(s) that exceeded %d retries", deadLettered, MaxRetries)
	}
}

func (actor *Actor) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-actor.jobs:
			if !ok {
				return
			}
			actor.process(ctx, job)
		}
	}
}

func (actor *Actor) process(ctx context.Context, job *jobstore.Job) {
	now := time.Now().UTC().Format(time.RFC3339)
	err := actor.jobStoreCommandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{
		JobID:     job.ID,
		Status:    jobstore.JobStatusRunning,
		UpdatedAt: now,
	})
	if err != nil {
		logger.Warnpf("reportsync: process: could not mark job %s started: %v", job.ID, err)
		return
	}
	downloadURL, err := actor.run(ctx, job)
	now = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		logger.Warnpf("reportsync: process: job %s failed: %v", job.ID, err)
		err := actor.jobStoreCommandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{
			JobID:      job.ID,
			Status:     jobstore.JobStatusFailed,
			FailReason: err.Error(),
			UpdatedAt:  now,
		})
		if err != nil {
			logger.Warnpf("reportsync: process: could not mark job %s failed: %v", job.ID, err)
		}
		return
	}
	err = actor.jobStoreCommandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{
		JobID:       job.ID,
		Status:      jobstore.JobStatusCompleted,
		DownloadURL: downloadURL,
		UpdatedAt:   now,
	})
	if err != nil {
		logger.Warnpf("reportsync: process: could not mark job %s completed: %v", job.ID, err)
	}
}

func (actor *Actor) run(ctx context.Context, job *jobstore.Job) (downloadURL string, err error) {
	switch job.Kind {
	case JobKindBacktest:
		return actor.runBacktest(ctx, job)
	default:
		return "", fmt.Errorf("unsupported job kind: %s", job.Kind)
	}
}
