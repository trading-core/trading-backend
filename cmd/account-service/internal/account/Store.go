package account

import (
	"context"
	"errors"

	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/config"
)

var ErrNotFound = errors.New("account not found")

type Store interface {
	Put(ctx context.Context, account Account) error
	Get(ctx context.Context, accountID string) (*Account, error)
}

type Account struct {
	ID            string          `json:"account_id"`
	Email         string          `json:"email,omitempty"`
	BrokerAccount *broker.Account `json:"broker_account,omitempty"`
}

func StoreFromEnv(ctx context.Context) Store {
	implementation := config.EnvString("ACCOUNT_STORE_IMPLEMENTATION", "INMEMORY")
	switch implementation {
	case "INMEMORY":
		return NewInMemoryStore()
	default:
		panic("unknown account store implementation: " + implementation)
	}
}
