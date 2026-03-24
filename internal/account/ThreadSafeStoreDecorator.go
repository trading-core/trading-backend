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

func (decorator *ThreadSafeStoreDecorator) Put(ctx context.Context, object *Object) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Put(ctx, object)
}

func (decorator *ThreadSafeStoreDecorator) Get(ctx context.Context, accountID string) (*Object, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, accountID)
}

func (decorator *ThreadSafeStoreDecorator) GetByEmail(ctx context.Context, email string) (*Object, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.GetByEmail(ctx, email)
}

func (decorator *ThreadSafeStoreDecorator) List(ctx context.Context) ([]*Object, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.List(ctx)
}
