package account

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

func (decorator *ThreadSafeStoreDecorator) Put(ctx context.Context, account Account) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Put(ctx, account)
}

func (decorator *ThreadSafeStoreDecorator) Get(ctx context.Context, accountID string) (*Account, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, accountID)
}
