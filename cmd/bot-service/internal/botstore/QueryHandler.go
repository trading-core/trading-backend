package botstore

import (
	"context"
)

type QueryHandler interface {
	Get(ctx context.Context, botID string) (*Bot, error)
	List(ctx context.Context) ([]*Bot, error)
}
