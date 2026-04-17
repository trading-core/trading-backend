package jobstore

import "context"

type ListInput struct {
	Page     int
	PageSize int
}

type ListResult struct {
	Jobs       []*Job `json:"jobs"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalCount int    `json:"total_count"`
	TotalPages int    `json:"total_pages"`
}

type QueryHandler interface {
	Get(ctx context.Context, jobID string) (*Job, error)
	// GetSystem fetches a job by ID without an ownership check; used by internal workers.
	GetSystem(ctx context.Context, jobID string) (*Job, error)
	// List returns a paginated page of jobs belonging to the user in ctx.
	List(ctx context.Context, input ListInput) (*ListResult, error)
}
