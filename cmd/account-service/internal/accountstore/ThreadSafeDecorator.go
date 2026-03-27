package accountstore

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

func (decorator *ThreadSafeDecorator) Create(ctx context.Context, input CreateInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Create(ctx, input)
}

func (decorator *ThreadSafeDecorator) LinkBrokerAccount(ctx context.Context, input LinkBrokerAccountInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.LinkBrokerAccount(ctx, input)
}

func (decorator *ThreadSafeDecorator) Get(ctx context.Context, input GetInput) (*Account, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, input)
}

func (decorator *ThreadSafeDecorator) List(ctx context.Context) ([]*Account, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.List(ctx)
}
