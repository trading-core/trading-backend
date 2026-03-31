package botstore

import (
	"context"
	"sync"
)

var _ QueryHandler = (*QueryHandlerThreadSafeDecorator)(nil)

type QueryHandlerThreadSafeDecorator struct {
	mutex     sync.Mutex
	decorated QueryHandler
}

type NewQueryHandlerThreadSafeDecoratorInput struct {
	Decorated QueryHandler
}

func NewQueryHandlerThreadSafeDecorator(input NewQueryHandlerThreadSafeDecoratorInput) *QueryHandlerThreadSafeDecorator {
	return &QueryHandlerThreadSafeDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *QueryHandlerThreadSafeDecorator) Get(ctx context.Context, botID string) (*Bot, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, botID)
}

func (decorator *QueryHandlerThreadSafeDecorator) List(ctx context.Context) ([]*Bot, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.List(ctx)
}
