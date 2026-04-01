package botsync

import "github.com/kduong/trading-backend/internal/eventsource"

const EventTypeBotDecisionRecorded eventsource.EventType = "bot_decision_recorded"

type EventFrame struct {
	eventsource.EventBase
	BotDecisionRecordedEvent *BotDecisionRecordedEvent `json:"bot_decision_recorded_event,omitempty"`
}

type BotDecisionRecordedEvent struct {
	BotID        string  `json:"bot_id"`
	Symbol       string  `json:"symbol"`
	StrategyType string  `json:"strategy_type"`
	Action       string  `json:"action"`
	Reason       string  `json:"reason"`
	Quantity     float64 `json:"quantity"`
	Price        float64 `json:"price"`
}
