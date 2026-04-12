package reportstore

import "github.com/kduong/trading-backend/internal/eventsource"

const (
	EventTypeReportEnqueued  eventsource.EventType = "report_enqueued"
	EventTypeReportStarted   eventsource.EventType = "report_started"
	EventTypeReportCompleted eventsource.EventType = "report_completed"
	EventTypeReportFailed    eventsource.EventType = "report_failed"
)

type EventFrame struct {
	eventsource.EventBase
	ReportEnqueuedEvent  *ReportEnqueuedEvent  `json:"report_enqueued_event,omitempty"`
	ReportStartedEvent   *ReportStartedEvent   `json:"report_started_event,omitempty"`
	ReportCompletedEvent *ReportCompletedEvent `json:"report_completed_event,omitempty"`
	ReportFailedEvent    *ReportFailedEvent    `json:"report_failed_event,omitempty"`
}

type ReportEnqueuedEvent struct {
	ReportID   string            `json:"report_id"`
	UserID     string            `json:"user_id"`
	Kind       string            `json:"kind"`
	Parameters map[string]string `json:"parameters,omitempty"`
	CreatedAt  string            `json:"created_at"`
}

type ReportStartedEvent struct {
	ReportID  string `json:"report_id"`
	UpdatedAt string `json:"updated_at"`
}

type ReportCompletedEvent struct {
	ReportID    string `json:"report_id"`
	DownloadURL string `json:"download_url"`
	UpdatedAt   string `json:"updated_at"`
}

type ReportFailedEvent struct {
	ReportID   string `json:"report_id"`
	FailReason string `json:"fail_reason"`
	UpdatedAt  string `json:"updated_at"`
}
