package account

import (
	"context"
	"sync"
)

var _ ObjectStore = (*LockingDecorator)(nil)

type LockingDecorator struct {
	mutex     sync.Mutex
	decorated ObjectStore
}

type NewLockingDecoratorInput struct {
	Decorated ObjectStore
}

func NewLockingDecorator(input NewLockingDecoratorInput) *LockingDecorator {
	return &LockingDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *LockingDecorator) GetObject(ctx context.Context, accountID string) (*Object, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.GetObject(ctx, accountID)
}
