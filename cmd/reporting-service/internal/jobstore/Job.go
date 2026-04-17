package jobstore

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

type Job struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	Name        string            `json:"name,omitempty"`
	Kind        string            `json:"kind"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Status      JobStatus         `json:"status"`
	FailReason  string            `json:"fail_reason,omitempty"`
	DownloadURL string            `json:"download_url,omitempty"`
	RetryCount  int               `json:"retry_count,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

func (job *Job) IsFinished() bool {
	switch job.Status {
	case JobStatusCompleted, JobStatusFailed:
		return true
	default:
		return false
	}
}
