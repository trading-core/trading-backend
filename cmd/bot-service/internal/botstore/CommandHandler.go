package botstore

import (
	"context"
)

type CommandHandler interface {
	Create(ctx context.Context, bot *Bot) error
	UpdateBotStatus(ctx context.Context, botID string, status BotStatus) error
	Delete(ctx context.Context, botID string) error
}
