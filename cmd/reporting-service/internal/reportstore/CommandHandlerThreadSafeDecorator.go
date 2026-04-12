package reportstore

import (
	"context"
	"sync"
)

var _ CommandHandler = (*CommandHandlerThreadSafeDecorator)(nil)

type CommandHandlerThreadSafeDecorator struct {
	mutex     sync.Mutex
	decorated CommandHandler
}

type NewCommandHandlerThreadSafeDecoratorInput struct {
	Decorated CommandHandler
}

func NewCommandHandlerThreadSafeDecorator(input NewCommandHandlerThreadSafeDecoratorInput) *CommandHandlerThreadSafeDecorator {
	return &CommandHandlerThreadSafeDecorator{decorated: input.Decorated}
}

func (decorator *CommandHandlerThreadSafeDecorator) Enqueue(ctx context.Context, report *Report) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Enqueue(ctx, report)
}

func (decorator *CommandHandlerThreadSafeDecorator) MarkStartedSystem(ctx context.Context, reportID string, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.MarkStartedSystem(ctx, reportID, updatedAt)
}

func (decorator *CommandHandlerThreadSafeDecorator) MarkCompletedSystem(ctx context.Context, reportID string, downloadURL string, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.MarkCompletedSystem(ctx, reportID, downloadURL, updatedAt)
}

func (decorator *CommandHandlerThreadSafeDecorator) MarkFailedSystem(ctx context.Context, reportID string, failReason string, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.MarkFailedSystem(ctx, reportID, failReason, updatedAt)
}

func (decorator *CommandHandlerThreadSafeDecorator) IncrementRetrySystem(ctx context.Context, reportID string, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.IncrementRetrySystem(ctx, reportID, updatedAt)
}
