package accountstore

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
	return &CommandHandlerThreadSafeDecorator{
		decorated: input.Decorated,
	}
}

func (decorator *CommandHandlerThreadSafeDecorator) Create(ctx context.Context, input CreateInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Create(ctx, input)
}

func (decorator *CommandHandlerThreadSafeDecorator) LinkBrokerAccount(ctx context.Context, input LinkBrokerAccountInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.LinkBrokerAccount(ctx, input)
}
