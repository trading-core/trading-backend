package pendingselectionstore

import (
	"time"

	"github.com/kduong/trading-backend/internal/broker"
)

type Store interface {
	Put(token string, entry Entry)
	Delete(token string)
	Get(token string) (Entry, bool)
}

type Entry struct {
	AccountID      string
	UserID         string
	Broker         broker.AccountType
	BrokerAccounts []string
	ExpiresAt      time.Time
}
