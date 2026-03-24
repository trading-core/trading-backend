package account

import (
	"context"
	"strings"
)

var _ Store = (*InMemoryStore)(nil)

type InMemoryStore struct {
	objectByAccountID map[string]*Object
	accountIDByEmail  map[string]string
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		objectByAccountID: make(map[string]*Object),
		accountIDByEmail:  make(map[string]string),
	}
}

func (store *InMemoryStore) Put(ctx context.Context, object *Object) error {
	if object == nil {
		return nil
	}
	email := strings.ToLower(strings.TrimSpace(object.Email))
	if len(email) > 0 {
		if existingAccountID, ok := store.accountIDByEmail[email]; ok && existingAccountID != object.AccountID {
			return ErrAccountAlreadyExists
		}
		store.accountIDByEmail[email] = object.AccountID
	}
	store.objectByAccountID[object.AccountID] = object
	return nil
}

func (store *InMemoryStore) Get(ctx context.Context, accountID string) (*Object, error) {
	object, ok := store.objectByAccountID[accountID]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return object, nil
}

func (store *InMemoryStore) GetByEmail(ctx context.Context, email string) (*Object, error) {
	accountID, ok := store.accountIDByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return store.Get(ctx, accountID)
}

func (store *InMemoryStore) List(ctx context.Context) ([]*Object, error) {
	output := make([]*Object, 0, len(store.objectByAccountID))
	for _, object := range store.objectByAccountID {
		output = append(output, object)
	}
	return output, nil
}
