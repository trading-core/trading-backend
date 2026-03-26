package account

import (
	"context"
	"errors"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/config"
)

var (
	ErrNotFound  = errors.New("account not found")
	ErrForbidden = errors.New("forbidden")
)

type Store interface {
	Put(ctx context.Context, account Account) error
	Get(ctx context.Context, accountID string) (*Account, error)
}

type Account struct {
	ID            string          `json:"account_id"`
	UserID        string          `json:"user_id"`
	Name          string          `json:"name"`
	BrokerLinked  bool            `json:"broker_linked"`
	BrokerAccount *broker.Account `json:"broker_account,omitempty"`
}

func StoreFromEnv() Store {
	implementation := config.EnvString("ACCOUNT_STORE_IMPLEMENTATION", "INMEMORY")
	switch implementation {
	case "INMEMORY":
		return NewInMemoryStore()
	default:
		panic("unknown account store implementation: " + implementation)
	}
}
