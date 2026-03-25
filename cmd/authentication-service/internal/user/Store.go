package user

import (
	"context"
	"errors"
	"time"

	"github.com/kduong/trading-backend/internal/config"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("user already exists")
)

type Store interface {
	Put(ctx context.Context, user User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type User struct {
	AccountID    string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

func StoreFromEnv(ctx context.Context) Store {
	implementation := config.EnvString("USER_STORE_IMPLEMENTATION", "INMEMORY")
	switch implementation {
	case "INMEMORY":
		return NewInMemoryStore()
	case "POSTGRES":
		dataSourceName := config.EnvStringOrFatal("USER_STORE_POSTGRES_DATASOURCE_NAME")
		return NewPostgresStore(ctx, dataSourceName)
	default:
		panic("unknown user store implementation: " + implementation)
	}
}
