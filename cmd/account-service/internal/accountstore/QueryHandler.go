package accountstore

import (
	"context"
)

type QueryHandler interface {
	Get(ctx context.Context, input GetInput) (*Account, error)
	List(ctx context.Context) ([]*Account, error)
}

type GetInput struct {
	AccountID string
}
