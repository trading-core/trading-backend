package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/lib/pq"
)

var _ Store = (*PostgresStore)(nil)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(ctx context.Context, dataSourceName string) *PostgresStore {
	db, err := sql.Open("postgres", dataSourceName)
	fatal.OnError(err)
	err = db.PingContext(ctx)
	fatal.OnError(err)
	store := &PostgresStore{db: db}
	err = store.ensureSchema(ctx)
	fatal.OnError(err)
	return store
}

func (store *PostgresStore) ensureSchema(ctx context.Context) error {
	_, err := store.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)
	`)
	return err
}

func (store *PostgresStore) Put(ctx context.Context, user User) error {
	_, err := store.db.ExecContext(
		ctx,
		`INSERT INTO users(id, email, password_hash, created_at) VALUES($1, $2, $3, $4)`,
		user.ID,
		strings.ToLower(strings.TrimSpace(user.Email)),
		user.PasswordHash,
		user.CreatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (store *PostgresStore) Get(ctx context.Context, userID string) (output *User, err error) {
	var user User
	err = store.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
			return
		}
		return
	}
	output = &user
	return
}

func (store *PostgresStore) GetByEmail(ctx context.Context, email string) (output *User, err error) {
	var user User
	err = store.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		strings.ToLower(strings.TrimSpace(email)),
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
			return
		}
		return
	}
	output = &user
	return
}
