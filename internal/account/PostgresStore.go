package account

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"
)

var _ Store = (*PostgresStore)(nil)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	store := &PostgresStore{db: db}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (store *PostgresStore) ensureSchema(ctx context.Context) error {
	_, err := store.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS auth_accounts (
			account_id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)
	`)
	return err
}

func (store *PostgresStore) Put(ctx context.Context, object *Object) error {
	_, err := store.db.ExecContext(
		ctx,
		`INSERT INTO auth_accounts(account_id, email, password_hash, created_at) VALUES($1, $2, $3, $4)`,
		object.AccountID,
		strings.ToLower(strings.TrimSpace(object.Email)),
		object.PasswordHash,
		object.CreatedAt,
	)
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return ErrAccountAlreadyExists
	}
	return err
}

func (store *PostgresStore) Get(ctx context.Context, accountID string) (*Object, error) {
	object := &Object{}
	err := store.db.QueryRowContext(
		ctx,
		`SELECT account_id, email, password_hash, created_at FROM auth_accounts WHERE account_id = $1`,
		accountID,
	).Scan(&object.AccountID, &object.Email, &object.PasswordHash, &object.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	return object, nil
}

func (store *PostgresStore) GetByEmail(ctx context.Context, email string) (*Object, error) {
	object := &Object{}
	err := store.db.QueryRowContext(
		ctx,
		`SELECT account_id, email, password_hash, created_at FROM auth_accounts WHERE email = $1`,
		strings.ToLower(strings.TrimSpace(email)),
	).Scan(&object.AccountID, &object.Email, &object.PasswordHash, &object.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	return object, nil
}

func (store *PostgresStore) List(ctx context.Context) ([]*Object, error) {
	rows, err := store.db.QueryContext(
		ctx,
		`SELECT account_id, email, password_hash, created_at FROM auth_accounts ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	output := make([]*Object, 0)
	for rows.Next() {
		object := &Object{}
		if err := rows.Scan(&object.AccountID, &object.Email, &object.PasswordHash, &object.CreatedAt); err != nil {
			return nil, err
		}
		output = append(output, object)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return output, nil
}
