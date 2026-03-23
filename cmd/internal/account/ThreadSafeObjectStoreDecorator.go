package account

import (
	"context"
	"sync"
)

var _ ObjectStore = (*ThreadSafeObjectStoreDecorator)(nil)

type ThreadSafeObjectStoreDecorator struct {
	mutex     sync.Mutex
	decorated ObjectStore
}

type NewThreadSafeObjectStoreDecoratorInput struct {
	Decorated ObjectStore
}

func NewThreadSafeObjectStoreDecorator(input NewThreadSafeObjectStoreDecoratorInput) *ThreadSafeObjectStoreDecorator {
	return &ThreadSafeObjectStoreDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *ThreadSafeObjectStoreDecorator) Put(ctx context.Context, object *Object) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Put(ctx, object)
}

func (decorator *ThreadSafeObjectStoreDecorator) Get(ctx context.Context, accountID string) (*Object, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, accountID)
}
