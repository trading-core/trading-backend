package account

import "context"

var _ Store = (*InMemoryStore)(nil)

type InMemoryStore struct {
	objectByAccountID map[string]*Object
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		objectByAccountID: make(map[string]*Object),
	}
}

func (store *InMemoryStore) Put(ctx context.Context, object *Object) error {
	store.objectByAccountID[object.AccountID] = object
	return nil
}

func (store *InMemoryStore) Get(ctx context.Context, accountID string) (*Object, error) {
	account, ok := store.objectByAccountID[accountID]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return account, nil
}
