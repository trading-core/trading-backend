package oauthstatestore

import (
	"time"

	"github.com/kduong/trading-backend/internal/broker"
)

type Store interface {
	Put(token string, entry Entry)
	Pop(token string) (Entry, bool)
}

type Entry struct {
	AccountID string
	UserID    string
	Broker    broker.AccountType
	ExpiresAt time.Time
}
