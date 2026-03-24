package account

import "time"

type BrokerType string

const (
	BrokerTypeTastyTrade BrokerType = "tastytrade"
)

type Object struct {
	AccountID       string     `json:"account_id"`
	Email           string     `json:"email,omitempty"`
	PasswordHash    string     `json:"-"`
	CreatedAt       time.Time  `json:"created_at,omitempty"`
	BrokerType      BrokerType `json:"broker_type,omitempty"`
	BrokerAccountID string     `json:"broker_account_id,omitempty"`
}
