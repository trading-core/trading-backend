package account

import (
	"context"
	"errors"
)

var ErrAccountNotFound = errors.New("account not found")

type ObjectStore interface {
	Put(ctx context.Context, object *Object) error
	Get(ctx context.Context, accountID string) (*Object, error)
}
