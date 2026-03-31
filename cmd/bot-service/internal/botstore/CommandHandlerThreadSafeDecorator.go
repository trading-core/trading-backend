package botstore

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

func (decorator *CommandHandlerThreadSafeDecorator) Create(ctx context.Context, bot *Bot) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Create(ctx, bot)
}

func (decorator *CommandHandlerThreadSafeDecorator) UpdateBotStatus(ctx context.Context, botID string, status BotStatus) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.UpdateBotStatus(ctx, botID, status)
}

func (decorator *CommandHandlerThreadSafeDecorator) Delete(ctx context.Context, botID string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Delete(ctx, botID)
}
