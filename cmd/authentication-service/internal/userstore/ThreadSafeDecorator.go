package userstore

import (
	"context"
	"sync"
)

var _ Store = (*ThreadSafeDecorator)(nil)

type ThreadSafeDecorator struct {
	mutex     sync.Mutex
	decorated Store
}

type NewThreadSafeDecoratorInput struct {
	Decorated Store
}

func NewThreadSafeDecorator(input NewThreadSafeDecoratorInput) *ThreadSafeDecorator {
	return &ThreadSafeDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *ThreadSafeDecorator) Put(ctx context.Context, user User) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Put(ctx, user)
}

func (decorator *ThreadSafeDecorator) GetByID(ctx context.Context, id string) (*User, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.GetByID(ctx, id)
}

func (decorator *ThreadSafeDecorator) GetByEmail(ctx context.Context, email string) (*User, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.GetByEmail(ctx, email)
}
