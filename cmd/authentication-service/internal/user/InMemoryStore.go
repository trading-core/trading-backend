package user

import (
	"context"
	"strings"
)

var _ Store = (*InMemoryStore)(nil)

type InMemoryStore struct {
	userByEmail map[string]User
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		userByEmail: make(map[string]User),
	}
}

func (store *InMemoryStore) Put(ctx context.Context, user User) error {
	if len(user.Email) > 0 {
		existingUser, hasEmail := store.userByEmail[user.Email]
		isEmailRegisteredToAnotherUser := hasEmail && existingUser.ID != user.ID
		if isEmailRegisteredToAnotherUser {
			return ErrAlreadyExists
		}
		store.userByEmail[user.Email] = user
	}
	return nil
}

func (store *InMemoryStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, ok := store.userByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, ErrNotFound
	}
	return &user, nil
}
