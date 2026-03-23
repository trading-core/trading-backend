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

func (store *InMemoryObjectStore) Put(ctx context.Context, object *Object) error {
	store.objectByAccountID[object.AccountID] = object
	return nil
}

func (store *InMemoryObjectStore) Get(ctx context.Context, accountID string) (*Object, error) {
	account, ok := store.objectByAccountID[accountID]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return account, nil
}
