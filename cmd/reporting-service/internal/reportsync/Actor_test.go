package reportsync_test

import (
	"context"
	"testing"
	"time"

	"github.com/kduong/trading-backend/cmd/reporting-service/internal/jobstore"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportsync"
	"github.com/kduong/trading-backend/internal/eventsource"
	. "github.com/smartystreets/goconvey/convey"
)

func newStore() (eventsource.Log, jobstore.CommandHandler, jobstore.QueryHandler) {
	log := eventsource.NewInMemoryLog("jobs")
	commandHandler := jobstore.NewEventSourcedCommandHandler(jobstore.NewEventSourcedCommandHandlerInput{Log: log})
	queryHandler := jobstore.NewEventSourcedQueryHandler(jobstore.NewEventSourcedQueryHandlerInput{Log: log})
	return log, commandHandler, queryHandler
}

func createJob(ctx context.Context, commandHandler jobstore.CommandHandler, jobID string) {
	now := time.Now().UTC().Format(time.RFC3339)
	err := commandHandler.CreateJob(ctx, &jobstore.Job{
		ID:        jobID,
		UserID:    "user-1",
		Kind:      "backtest",
		Status:    jobstore.JobStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	})
	So(err, ShouldBeNil)
}

// newTestActor creates an Actor wired to the given store with no real backtest
// dependencies; it is used only for testing the catchup/recovery phases.
func newTestActor(log eventsource.Log, commandHandler jobstore.CommandHandler) *reportsync.Actor {
	return reportsync.NewActor(reportsync.NewActorInput{
		CommandHandler: commandHandler,
		Log:            log,
	})
}

func TestActorCatchUp(t *testing.T) {
	Convey("Given an actor", t, func() {
		ctx := context.Background()
		log, commandHandler, _ := newStore()
		actor := newTestActor(log, commandHandler)

		Convey("When there are no jobs, catchup succeeds without error", func() {
			actor.CatchUp(ctx)

			So(actor.JobsLen(), ShouldEqual, 0)
		})

		Convey("When jobs exist, catchup populates the actor map for recovery", func() {
			createJob(ctx, commandHandler, "job-1")

			actor.CatchUp(ctx)
			actor.CompleteCatchup(ctx)

			So(actor.JobsLen(), ShouldEqual, 1)
		})
	})
}

func TestActorRecover(t *testing.T) {
	Convey("Given an actor", t, func() {
		ctx := context.Background()
		log, commandHandler, queryHandler := newStore()
		actor := newTestActor(log, commandHandler)

		Convey("When there are no interrupted jobs", func() {
			actor.CatchUp(ctx)
			actor.CompleteCatchup(ctx)

			Convey("Then no jobs are queued", func() {
				So(actor.JobsLen(), ShouldEqual, 0)
			})
		})

		Convey("When a pending job has not yet been retried", func() {
			createJob(ctx, commandHandler, "job-1")

			actor.CatchUp(ctx)
			actor.CompleteCatchup(ctx)

			Convey("Then the job is requeued with retry count 1", func() {
				So(actor.JobsLen(), ShouldEqual, 1)

				job, err := queryHandler.GetSystem(ctx, "job-1")
				So(err, ShouldBeNil)
				So(job.RetryCount, ShouldEqual, 1)
				So(job.Status, ShouldEqual, jobstore.JobStatusPending)
			})
		})

		Convey("When a running job is recovered", func() {
			createJob(ctx, commandHandler, "job-2")
			now := time.Now().UTC().Format(time.RFC3339)
			So(commandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{JobID: "job-2", Status: jobstore.JobStatusRunning, UpdatedAt: now}), ShouldBeNil)

			actor.CatchUp(ctx)
			actor.CompleteCatchup(ctx)

			Convey("Then the job is requeued with retry count 1", func() {
				So(actor.JobsLen(), ShouldEqual, 1)

				job, err := queryHandler.GetSystem(ctx, "job-2")
				So(err, ShouldBeNil)
				So(job.RetryCount, ShouldEqual, 1)
				So(job.Status, ShouldEqual, jobstore.JobStatusPending)
			})
		})

		Convey("When a job has been retried twice and is interrupted again", func() {
			createJob(ctx, commandHandler, "job-3")
			now := time.Now().UTC().Format(time.RFC3339)
			So(commandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{JobID: "job-3", Status: jobstore.JobStatusPending, RetryCount: 1, UpdatedAt: now}), ShouldBeNil)
			So(commandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{JobID: "job-3", Status: jobstore.JobStatusPending, RetryCount: 2, UpdatedAt: now}), ShouldBeNil)

			actor.CatchUp(ctx)
			actor.CompleteCatchup(ctx)

			Convey("Then the job is requeued with retry count 3", func() {
				So(actor.JobsLen(), ShouldEqual, 1)

				job, err := queryHandler.GetSystem(ctx, "job-3")
				So(err, ShouldBeNil)
				So(job.RetryCount, ShouldEqual, 3)
				So(job.Status, ShouldEqual, jobstore.JobStatusPending)
			})
		})

		Convey("When a job has already hit the retry limit", func() {
			createJob(ctx, commandHandler, "job-4")
			now := time.Now().UTC().Format(time.RFC3339)
			So(commandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{JobID: "job-4", Status: jobstore.JobStatusPending, RetryCount: 1, UpdatedAt: now}), ShouldBeNil)
			So(commandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{JobID: "job-4", Status: jobstore.JobStatusPending, RetryCount: 2, UpdatedAt: now}), ShouldBeNil)
			So(commandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{JobID: "job-4", Status: jobstore.JobStatusPending, RetryCount: 3, UpdatedAt: now}), ShouldBeNil)

			actor.CatchUp(ctx)
			actor.CompleteCatchup(ctx)

			Convey("Then the job is dead-lettered and not requeued", func() {
				So(actor.JobsLen(), ShouldEqual, 0)

				job, err := queryHandler.GetSystem(ctx, "job-4")
				So(err, ShouldBeNil)
				So(job.Status, ShouldEqual, jobstore.JobStatusFailed)
				So(job.FailReason, ShouldContainSubstring, "dead letter")
			})
		})

		Convey("When a completed job exists", func() {
			createJob(ctx, commandHandler, "job-5")
			now := time.Now().UTC().Format(time.RFC3339)
			So(commandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{JobID: "job-5", Status: jobstore.JobStatusRunning, UpdatedAt: now}), ShouldBeNil)
			So(commandHandler.UpdateJobStatus(ctx, jobstore.UpdateJobStatusInput{JobID: "job-5", Status: jobstore.JobStatusCompleted, DownloadURL: "/download/job-5", UpdatedAt: now}), ShouldBeNil)

			actor.CatchUp(ctx)
			actor.CompleteCatchup(ctx)

			Convey("Then the completed job is left untouched", func() {
				So(actor.JobsLen(), ShouldEqual, 0)

				job, err := queryHandler.GetSystem(ctx, "job-5")
				So(err, ShouldBeNil)
				So(job.Status, ShouldEqual, jobstore.JobStatusCompleted)
			})
		})
	})
}
