package accountstore

import (
	"context"
	"errors"

	"github.com/kduong/trading-backend/internal/broker"
)

var (
	ErrAccountNotFound            = errors.New("account not found")
	ErrAccountForbidden           = errors.New("account forbidden")
	ErrBrokerAccountAlreadyLinked = errors.New("broker account already linked")
)

type Store interface {
	Create(ctx context.Context, input CreateInput) error
	LinkBrokerAccount(ctx context.Context, input LinkBrokerAccountInput) error
	Get(ctx context.Context, input GetInput) (*Account, error)
	List(ctx context.Context) ([]*Account, error)
}

type CreateInput struct {
	AccountID   string
	AccountName string
}

type LinkBrokerAccountInput struct {
	AccountID     string
	BrokerAccount *broker.Account
}

type GetInput struct {
	AccountID string
}

type Account struct {
	ID            string          `json:"account_id"`
	UserID        string          `json:"user_id"`
	Name          string          `json:"name"`
	BrokerLinked  bool            `json:"broker_linked"`
	BrokerAccount *broker.Account `json:"broker_account,omitempty"`
}
