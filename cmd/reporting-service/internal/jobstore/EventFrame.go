package jobstore

import "github.com/kduong/trading-backend/internal/eventsource"

const (
	EventTypeJobEnqueued  eventsource.EventType = "job_enqueued"
	EventTypeJobStarted   eventsource.EventType = "job_started"
	EventTypeJobCompleted eventsource.EventType = "job_completed"
	EventTypeJobFailed    eventsource.EventType = "job_failed"
	EventTypeJobRetried   eventsource.EventType = "job_retried"
)

type EventFrame struct {
	eventsource.EventBase
	JobEnqueuedEvent  *JobEnqueuedEvent  `json:"job_enqueued_event,omitempty"`
	JobStartedEvent   *JobStartedEvent   `json:"job_started_event,omitempty"`
	JobCompletedEvent *JobCompletedEvent `json:"job_completed_event,omitempty"`
	JobFailedEvent    *JobFailedEvent    `json:"job_failed_event,omitempty"`
	JobRetriedEvent   *JobRetriedEvent   `json:"job_retried_event,omitempty"`
}

type JobEnqueuedEvent struct {
	JobID      string            `json:"job_id"`
	UserID     string            `json:"user_id"`
	Name       string            `json:"name,omitempty"`
	Kind       string            `json:"kind"`
	Parameters map[string]string `json:"parameters,omitempty"`
	CreatedAt  string            `json:"created_at"`
}

type JobStartedEvent struct {
	JobID     string `json:"job_id"`
	UpdatedAt string `json:"updated_at"`
}

type JobCompletedEvent struct {
	JobID       string `json:"job_id"`
	DownloadURL string `json:"download_url"`
	UpdatedAt   string `json:"updated_at"`
}

type JobFailedEvent struct {
	JobID      string `json:"job_id"`
	FailReason string `json:"fail_reason"`
	UpdatedAt  string `json:"updated_at"`
}

type JobRetriedEvent struct {
	JobID      string `json:"job_id"`
	RetryCount int    `json:"retry_count"`
	UpdatedAt  string `json:"updated_at"`
}
