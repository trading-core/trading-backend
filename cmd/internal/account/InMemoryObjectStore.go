package account

import "context"

var _ ObjectStore = (*InMemoryObjectStore)(nil)

type InMemoryObjectStore struct {
	objectByAccountID map[string]*Object
}

func NewInMemoryObjectStore() *InMemoryObjectStore {
	return &InMemoryObjectStore{
		objectByAccountID: make(map[string]*Object),
	}
}

func (store *InMemoryObjectStore) GetObject(ctx context.Context, accountID string) (*Object, error) {
	account, ok := store.objectByAccountID[accountID]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return account, nil
}
