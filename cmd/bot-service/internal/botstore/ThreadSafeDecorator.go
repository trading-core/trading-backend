package botstore

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

func (decorator *ThreadSafeDecorator) Create(ctx context.Context, bot *Bot) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Create(ctx, bot)
}

func (decorator *ThreadSafeDecorator) UpdateBotStatus(ctx context.Context, botID string, status BotStatus) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.UpdateBotStatus(ctx, botID, status)
}

func (decorator *ThreadSafeDecorator) Get(ctx context.Context, botID string) (*Bot, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, botID)
}

func (decorator *ThreadSafeDecorator) List(ctx context.Context) ([]*Bot, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.List(ctx)
}

func (decorator *ThreadSafeDecorator) Delete(ctx context.Context, botID string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Delete(ctx, botID)
}
