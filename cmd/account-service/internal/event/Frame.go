package event

import "github.com/kduong/trading-backend/internal/eventsource"

const EventTypeAccountCreated eventsource.EventType = "account_created"

type Frame struct {
	eventsource.EventBase
	AccountCreatedEvent *AccountCreatedEvent `json:"account_created_event,omitempty"`
}

type AccountCreatedEvent struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	UserID      string `json:"user_id"`
}
