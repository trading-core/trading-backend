package accountstore

import "github.com/kduong/trading-backend/internal/broker"

type Account struct {
	ID            string          `json:"account_id"`
	UserID        string          `json:"user_id"`
	Name          string          `json:"name"`
	BrokerLinked  bool            `json:"broker_linked"`
	BrokerAccount *broker.Account `json:"broker_account,omitempty"`
}
