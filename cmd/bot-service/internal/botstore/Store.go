package botstore

import (
	"context"
	"errors"
)

var (
	ErrBotAlreadyExists = errors.New("bot already exists")
	ErrBotNotFound      = errors.New("bot not found")
	ErrBotForbidden     = errors.New("bot forbidden")
)

type Store interface {
	Create(ctx context.Context, bot *Bot) error
	UpdateBotStatus(ctx context.Context, botID string, status BotStatus) error
	Get(ctx context.Context, botID string) (*Bot, error)
	List(ctx context.Context) ([]*Bot, error)
	Delete(ctx context.Context, botID string) error
}

type BotStatus string

const (
	BotStatusRunning BotStatus = "running"
	BotStatusStopped BotStatus = "stopped"
)

type Bot struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	AccountID       string    `json:"account_id"`
	BrokerAccountID string    `json:"broker_account_id,omitempty"`
	BrokerType      string    `json:"broker_type,omitempty"`
	Name            string    `json:"name"`
	Status          BotStatus `json:"status"`
	CreatedAt       string    `json:"created_at"`
}
