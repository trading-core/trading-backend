package accountstore

import (
	"github.com/kduong/trading-backend/internal/broker"
	"github.com/kduong/trading-backend/internal/eventsource"
)

const (
	EventTypeAccountCreated      eventsource.EventType = "account_created"
	EventTypeBrokerAccountLinked eventsource.EventType = "broker_account_linked"
)

type EventFrame struct {
	eventsource.EventBase
	AccountCreatedEvent      *AccountCreatedEvent      `json:"account_created_event,omitempty"`
	BrokerAccountLinkedEvent *BrokerAccountLinkedEvent `json:"broker_account_linked_event,omitempty"`
}

type AccountCreatedEvent struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	UserID      string `json:"user_id"`
}

type BrokerAccountLinkedEvent struct {
	AccountID     string          `json:"account_id"`
	BrokerAccount *broker.Account `json:"broker_account"`
}
