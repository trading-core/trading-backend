package recoveryworker_test

import (
	"context"
	"testing"
	"time"

	"github.com/kduong/trading-backend/cmd/reporting-service/internal/recoveryworker"
	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportstore"
	"github.com/kduong/trading-backend/internal/eventsource"
	. "github.com/smartystreets/goconvey/convey"
)

func newStore() (reportstore.CommandHandler, reportstore.QueryHandler) {
	log := eventsource.NewInMemoryLog("reports")
	commandHandler := reportstore.NewEventSourcedCommandHandler(reportstore.NewEventSourcedCommandHandlerInput{Log: log})
	queryHandler := reportstore.NewEventSourcedQueryHandler(reportstore.NewEventSourcedQueryHandlerInput{Log: log})
	return commandHandler, queryHandler
}

func enqueueReport(ctx context.Context, commandHandler reportstore.CommandHandler, reportID string) {
	now := time.Now().UTC().Format(time.RFC3339)
	err := commandHandler.Enqueue(ctx, &reportstore.Report{
		ID:        reportID,
		UserID:    "user-1",
		Kind:      "backtest",
		Status:    reportstore.ReportStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	})
	So(err, ShouldBeNil)
}

func TestRecover(t *testing.T) {
	Convey("Given a recovery worker", t, func() {
		ctx := context.Background()
		commandHandler, queryHandler := newStore()
		jobs := make(chan string, 10)
		worker := recoveryworker.New(recoveryworker.WorkerInput{
			CommandHandler: commandHandler,
			QueryHandler:   queryHandler,
			Jobs:           jobs,
		})

		Convey("When there are no interrupted reports", func() {
			worker.Recover(ctx)

			Convey("Then no jobs are queued", func() {
				So(jobs, ShouldBeEmpty)
			})
		})

		Convey("When a pending report has not yet been retried", func() {
			enqueueReport(ctx, commandHandler, "report-1")

			worker.Recover(ctx)

			Convey("Then the report is requeued with retry count 1", func() {
				So(len(jobs), ShouldEqual, 1)
				So(<-jobs, ShouldEqual, "report-1")

				reports, err := queryHandler.ListAll(ctx)
				So(err, ShouldBeNil)
				So(reports[0].RetryCount, ShouldEqual, 1)
				So(reports[0].Status, ShouldEqual, reportstore.ReportStatusPending)
			})
		})

		Convey("When a running report is recovered", func() {
			enqueueReport(ctx, commandHandler, "report-2")
			now := time.Now().UTC().Format(time.RFC3339)
			So(commandHandler.MarkStartedSystem(ctx, "report-2", now), ShouldBeNil)

			worker.Recover(ctx)

			Convey("Then the report is requeued with retry count 1", func() {
				So(len(jobs), ShouldEqual, 1)
				So(<-jobs, ShouldEqual, "report-2")

				reports, err := queryHandler.ListAll(ctx)
				So(err, ShouldBeNil)
				So(reports[0].RetryCount, ShouldEqual, 1)
				So(reports[0].Status, ShouldEqual, reportstore.ReportStatusPending)
			})
		})

		Convey("When a report has been retried twice and is interrupted again", func() {
			enqueueReport(ctx, commandHandler, "report-3")
			now := time.Now().UTC().Format(time.RFC3339)
			So(commandHandler.IncrementRetrySystem(ctx, "report-3", now), ShouldBeNil)
			So(commandHandler.IncrementRetrySystem(ctx, "report-3", now), ShouldBeNil)

			worker.Recover(ctx)

			Convey("Then the report is requeued with retry count 3", func() {
				So(len(jobs), ShouldEqual, 1)
				So(<-jobs, ShouldEqual, "report-3")

				reports, err := queryHandler.ListAll(ctx)
				So(err, ShouldBeNil)
				So(reports[0].RetryCount, ShouldEqual, 3)
				So(reports[0].Status, ShouldEqual, reportstore.ReportStatusPending)
			})
		})

		Convey("When a report has already hit the retry limit", func() {
			enqueueReport(ctx, commandHandler, "report-4")
			now := time.Now().UTC().Format(time.RFC3339)
			So(commandHandler.IncrementRetrySystem(ctx, "report-4", now), ShouldBeNil)
			So(commandHandler.IncrementRetrySystem(ctx, "report-4", now), ShouldBeNil)
			So(commandHandler.IncrementRetrySystem(ctx, "report-4", now), ShouldBeNil)

			worker.Recover(ctx)

			Convey("Then the report is dead-lettered and not requeued", func() {
				So(jobs, ShouldBeEmpty)

				reports, err := queryHandler.ListAll(ctx)
				So(err, ShouldBeNil)
				So(reports[0].Status, ShouldEqual, reportstore.ReportStatusFailed)
				So(reports[0].FailReason, ShouldContainSubstring, "dead letter")
			})
		})

		Convey("When a completed report exists", func() {
			enqueueReport(ctx, commandHandler, "report-5")
			now := time.Now().UTC().Format(time.RFC3339)
			So(commandHandler.MarkStartedSystem(ctx, "report-5", now), ShouldBeNil)
			So(commandHandler.MarkCompletedSystem(ctx, "report-5", "/download/report-5", now), ShouldBeNil)

			worker.Recover(ctx)

			Convey("Then the completed report is left untouched", func() {
				So(jobs, ShouldBeEmpty)

				reports, err := queryHandler.ListAll(ctx)
				So(err, ShouldBeNil)
				So(reports[0].Status, ShouldEqual, reportstore.ReportStatusCompleted)
			})
		})
	})
}
