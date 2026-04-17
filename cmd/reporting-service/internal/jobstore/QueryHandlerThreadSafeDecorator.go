package jobstore

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
	return &QueryHandlerThreadSafeDecorator{decorated: input.Decorated}
}

func (decorator *QueryHandlerThreadSafeDecorator) Get(ctx context.Context, jobID string) (*Job, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, jobID)
}

func (decorator *QueryHandlerThreadSafeDecorator) GetSystem(ctx context.Context, jobID string) (*Job, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.GetSystem(ctx, jobID)
}

func (decorator *QueryHandlerThreadSafeDecorator) List(ctx context.Context, input ListInput) (*ListResult, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.List(ctx, input)
}
