package account

import (
	"context"

	"github.com/kduong/trading-backend/internal/contextx"
)

var _ Store = (*InMemoryStore)(nil)

type InMemoryStore struct {
	accountByID map[string]Account
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		accountByID: make(map[string]Account),
	}
}

func (store *InMemoryStore) Put(ctx context.Context, account Account) error {
	store.accountByID[account.ID] = account
	return nil
}

func (store *InMemoryStore) Get(ctx context.Context, accountID string) (*Account, error) {
	account, ok := store.accountByID[accountID]
	if !ok {
		return nil, ErrNotFound
	}
	userID := contextx.GetUserID(ctx)
	if account.UserID != userID {
		return nil, ErrForbidden
	}
	return &account, nil
}
