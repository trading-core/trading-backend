package botstore

import "github.com/kduong/trading-backend/internal/eventsource"

const (
	EventTypeBotCreated       eventsource.EventType = "bot_created"
	EventTypeBotStatusUpdated eventsource.EventType = "bot_status_updated"
	EventTypeBotStatusDeleted eventsource.EventType = "bot_status_deleted"
)

type EventFrame struct {
	eventsource.EventBase
	BotCreatedEvent       *BotCreatedEvent       `json:"bot_created_event,omitempty"`
	BotStatusUpdatedEvent *BotStatusUpdatedEvent `json:"bot_status_updated_event,omitempty"`
	BotStatusDeletedEvent *BotStatusDeletedEvent `json:"bot_status_deleted_event,omitempty"`
}

type BotCreatedEvent struct {
	BotID             string    `json:"bot_id"`
	UserID            string    `json:"user_id"`
	AccountID         string    `json:"account_id"`
	BrokerAccountID   string    `json:"broker_account_id"`
	BrokerType        string    `json:"broker_type"`
	Symbol            string    `json:"symbol"`
	StrategyTradeType string    `json:"strategy_trade_type"`
	AllocationPercent float64   `json:"allocation_percent"`
	Status            BotStatus `json:"status"`
	CreatedAt         string    `json:"created_at"`
}

type BotStatusUpdatedEvent struct {
	BotID  string    `json:"bot_id"`
	Status BotStatus `json:"status"`
}

type BotStatusDeletedEvent struct {
	BotID string `json:"bot_id"`
}
