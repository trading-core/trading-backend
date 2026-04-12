package reportstore

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusRunning   ReportStatus = "running"
	ReportStatusCompleted ReportStatus = "completed"
	ReportStatusFailed    ReportStatus = "failed"
)

type Report struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	Name        string            `json:"name,omitempty"`
	Kind        string            `json:"kind"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Status      ReportStatus      `json:"status"`
	FailReason  string            `json:"fail_reason,omitempty"`
	DownloadURL string            `json:"download_url,omitempty"`
	RetryCount  int               `json:"retry_count,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}
