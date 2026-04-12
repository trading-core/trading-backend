// Package recoveryworker implements the dead-letter queue mechanism for the
// reporting service. On startup it scans the event log for any reports that
// were left in a non-terminal state (pending or running) due to a previous
// crash, and requeues them so the backtest worker can pick them up again.
//
// Reports that were "running" when the service died are first marked failed
// (they may have produced partial output), then re-enqueued as fresh pending
// jobs so the worker retries them cleanly.
package recoveryworker

import (
	"context"
	"time"

	"github.com/kduong/trading-backend/cmd/reporting-service/internal/reportstore"
	"github.com/kduong/trading-backend/internal/logger"
)

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
	for _, report := range reports {
		switch report.Status {
		case reportstore.ReportStatusRunning:
			// The service crashed mid-run. Mark it failed then requeue.
			now := time.Now().UTC().Format(time.RFC3339)
			if err := worker.commandHandler.MarkFailedSystem(ctx, report.ID, "service restarted during execution", now); err != nil {
				logger.Warnpf("recoveryworker: could not mark report %s failed: %v", report.ID, err)
				continue
			}
			worker.requeue(ctx, report.ID)
			recovered++

		case reportstore.ReportStatusPending:
			// Enqueued but never picked up (e.g. service crashed before worker read it).
			worker.requeue(ctx, report.ID)
			recovered++
		}
	}

	if recovered > 0 {
		logger.Warnpf("recoveryworker: requeued %d interrupted report(s)", recovered)
	}
}

func (worker *Worker) requeue(ctx context.Context, reportID string) {
	select {
	case worker.jobs <- reportID:
	case <-ctx.Done():
	}
}
