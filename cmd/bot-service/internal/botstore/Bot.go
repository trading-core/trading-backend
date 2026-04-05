package botstore

import "github.com/kduong/trading-backend/internal/tradingstrategy"

type BotStatus string

const (
	BotStatusRunning BotStatus = "running"
	BotStatusStopped BotStatus = "stopped"
)

type Bot struct {
	ID                string                          `json:"id"`
	UserID            string                          `json:"user_id"`
	AccountID         string                          `json:"account_id"`
	BrokerAccountID   string                          `json:"broker_account_id,omitempty"`
	BrokerType        string                          `json:"broker_type,omitempty"`
	Symbol            string                          `json:"symbol"`
	AllocationPercent float64                         `json:"allocation_percent"`
	ScalpingParams    *tradingstrategy.ScalpingParams `json:"scalping_params,omitempty"`
	Status            BotStatus                       `json:"status"`
	CreatedAt         string                          `json:"created_at"`
}
