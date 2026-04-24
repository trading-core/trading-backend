package entrystore

import "context"

type ListInput struct {
	From     string
	To       string
	Page     int
	PageSize int
}

type ListResult struct {
	Entries    []*Entry `json:"entries"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	TotalCount int      `json:"total_count"`
	TotalPages int      `json:"total_pages"`
}

type QueryHandler interface {
	Get(ctx context.Context, date string) (*Entry, error)
	List(ctx context.Context, input ListInput) (*ListResult, error)
}
