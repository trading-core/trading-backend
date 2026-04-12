// Package recoveryworker implements the dead-letter queue mechanism for the
// reporting service. On startup it scans the event log for any reports that
// were left in a non-terminal state (pending or running) due to a previous
// crash, and requeues them so the backtest worker can pick them up again.
//
// Each interrupted report is given up to maxRetries attempts. On each recovery
// its retry count is incremented and it is requeued as pending. Once the retry
// count reaches maxRetries the report is permanently failed with a dead-letter
// message instead of being requeued.
package recoveryworker

import (
	"context"
	"fmt"
	"time"

	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportstore"
	"github.com/kduong/trading-backend/internal/logger"
)

const maxRetries = 3

type WorkerInput struct {
	CommandHandler reportstore.CommandHandler
	QueryHandler   reportstore.QueryHandler
	// Jobs is the channel the backtest worker reads from; recovered reports
	// are pushed here after being reset to pending.
	Jobs chan<- string
}

type Worker struct {
	commandHandler reportstore.CommandHandler
	queryHandler   reportstore.QueryHandler
	jobs           chan<- string
}

func New(input WorkerInput) *Worker {
	return &Worker{
		commandHandler: input.CommandHandler,
		queryHandler:   input.QueryHandler,
		jobs:           input.Jobs,
	}
}

// Recover scans all reports visible to the system and requeues any that are
// stuck in a non-terminal state. It must be called once before the backtest
// worker starts consuming jobs.
func (worker *Worker) Recover(ctx context.Context) {
	reports, err := worker.queryHandler.ListAll(ctx)
	if err != nil {
		logger.Warnpf("recoveryworker: could not list reports for recovery: %v", err)
		return
	}

	recovered := 0
	deadLettered := 0
	for _, report := range reports {
		switch report.Status {
		case reportstore.ReportStatusRunning, reportstore.ReportStatusPending:
			now := time.Now().UTC().Format(time.RFC3339)
			nextRetry := report.RetryCount + 1
			if nextRetry > maxRetries {
				reason := fmt.Sprintf("dead letter: job failed to complete after %d attempts", maxRetries)
				if err := worker.commandHandler.MarkFailedSystem(ctx, report.ID, reason, now); err != nil {
					logger.Warnpf("recoveryworker: could not dead-letter report %s: %v", report.ID, err)
				} else {
					deadLettered++
				}
				continue
			}
			if err := worker.commandHandler.IncrementRetrySystem(ctx, report.ID, now); err != nil {
				logger.Warnpf("recoveryworker: could not increment retry for report %s: %v", report.ID, err)
				continue
			}
			worker.requeue(ctx, report.ID)
			recovered++
		}
	}

	if recovered > 0 {
		logger.Warnpf("recoveryworker: requeued %d interrupted report(s)", recovered)
	}
	if deadLettered > 0 {
		logger.Warnpf("recoveryworker: dead-lettered %d report(s) that exceeded %d retries", deadLettered, maxRetries)
	}
}

func (worker *Worker) requeue(ctx context.Context, reportID string) {
	select {
	case worker.jobs <- reportID:
	case <-ctx.Done():
	}
}
