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

func (decorator *ThreadSafeStoreDecorator) Create(ctx context.Context, input CreateInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Create(ctx, input)
}

func (decorator *ThreadSafeStoreDecorator) LinkBrokerAccount(ctx context.Context, input LinkBrokerAccountInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.LinkBrokerAccount(ctx, input)
}

func (decorator *ThreadSafeStoreDecorator) Get(ctx context.Context, input GetInput) (*Account, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, input)
}

func (decorator *ThreadSafeStoreDecorator) List(ctx context.Context) ([]*Account, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.List(ctx)
}
