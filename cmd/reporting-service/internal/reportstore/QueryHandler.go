package reportstore

import "context"

type ListInput struct {
	Page     int
	PageSize int
}

type ListResult struct {
	Reports    []*Report `json:"reports"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalCount int       `json:"total_count"`
	TotalPages int       `json:"total_pages"`
}

type QueryHandler interface {
	Get(ctx context.Context, reportID string) (*Report, error)
	// GetSystem fetches a report by ID without an ownership check; used by internal workers.
	GetSystem(ctx context.Context, reportID string) (*Report, error)
	// List returns a paginated page of reports belonging to the user in ctx.
	List(ctx context.Context, input ListInput) (*ListResult, error)
	// ListAll returns every report regardless of owner; used by the recovery worker.
	ListAll(ctx context.Context) ([]*Report, error)
}
