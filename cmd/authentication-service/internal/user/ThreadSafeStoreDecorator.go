package user

import (
	"context"
	"sync"
)

var _ Store = (*ThreadSafeStoreDecorator)(nil)

type ThreadSafeStoreDecorator struct {
	mutex     sync.Mutex
	decorated Store
}

type NewThreadSafeStoreDecoratorInput struct {
	Decorated Store
}

func NewThreadSafeStoreDecorator(input NewThreadSafeStoreDecoratorInput) *ThreadSafeStoreDecorator {
	return &ThreadSafeStoreDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *ThreadSafeStoreDecorator) Put(ctx context.Context, user User) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Put(ctx, user)
}

func (decorator *ThreadSafeStoreDecorator) GetByEmail(ctx context.Context, email string) (*User, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.GetByEmail(ctx, email)
}
