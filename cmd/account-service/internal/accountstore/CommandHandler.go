package accountstore

import (
	"context"

	"github.com/kduong/trading-backend/internal/broker"
)

type CommandHandler interface {
	Create(ctx context.Context, input CreateInput) error
	LinkBrokerAccount(ctx context.Context, input LinkBrokerAccountInput) error
}

type CreateInput struct {
	AccountID   string
	AccountName string
}

type LinkBrokerAccountInput struct {
	AccountID     string
	BrokerAccount *broker.Account
}
