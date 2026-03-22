package account

import (
	"context"
	"errors"
)

var ErrAccountNotFound = errors.New("account not found")

type ObjectStore interface {
	GetObject(ctx context.Context, accountID string) (*Object, error)
}
