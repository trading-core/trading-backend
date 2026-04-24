package entrystore

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

func (decorator *CommandHandlerThreadSafeDecorator) UpsertEntry(ctx context.Context, entry *Entry) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.UpsertEntry(ctx, entry)
}

func (decorator *CommandHandlerThreadSafeDecorator) DeleteEntry(ctx context.Context, input DeleteEntryInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.DeleteEntry(ctx, input)
}
