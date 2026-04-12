package reportstore

import "context"

type CommandHandler interface {
	Enqueue(ctx context.Context, report *Report) error
	// System-scoped: used by internal workers that have no user context.
	MarkStartedSystem(ctx context.Context, reportID string, updatedAt string) error
	MarkCompletedSystem(ctx context.Context, reportID string, downloadURL string, updatedAt string) error
	MarkFailedSystem(ctx context.Context, reportID string, failReason string, updatedAt string) error
	IncrementRetrySystem(ctx context.Context, reportID string, updatedAt string) error
}
