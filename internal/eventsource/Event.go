package eventsource

import "time"

type EventType string

type EventBase struct {
	Type            EventType `json:"type"`
	TimestampMillis int64     `json:"timestamp_millis"`
}

func NewEventBase(eventType EventType) EventBase {
	return EventBase{
		Type:            eventType,
		TimestampMillis: time.Now().UnixMilli(),
	}
}
