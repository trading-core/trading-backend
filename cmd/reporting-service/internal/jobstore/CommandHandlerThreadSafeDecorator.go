package jobstore

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

func (decorator *CommandHandlerThreadSafeDecorator) CreateJob(ctx context.Context, job *Job) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.CreateJob(ctx, job)
}

func (decorator *CommandHandlerThreadSafeDecorator) UpdateJobStatus(ctx context.Context, input UpdateJobStatusInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.UpdateJobStatus(ctx, input)
}
