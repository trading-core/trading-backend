package account

import (
	"context"
	"errors"
)

var ErrAccountNotFound = errors.New("account not found")
var ErrAccountAlreadyExists = errors.New("account already exists")

type Store interface {
	Put(ctx context.Context, object *Object) error
	Get(ctx context.Context, accountID string) (*Object, error)
	GetByEmail(ctx context.Context, email string) (*Object, error)
	List(ctx context.Context) ([]*Object, error)
}
