package jobstore

import "context"

type CommandHandler interface {
	CreateJob(ctx context.Context, job *Job) error
	UpdateJobStatus(ctx context.Context, input UpdateJobStatusInput) error
}

type UpdateJobStatusInput struct {
	JobID       string
	Status      JobStatus
	DownloadURL string // used when Status == JobStatusCompleted
	FailReason  string // used when Status == JobStatusFailed
	RetryCount  int    // used when Status == JobStatusPending (retry)
	UpdatedAt   string
}
