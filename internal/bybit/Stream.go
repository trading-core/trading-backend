package bybit

import (
	"context"
)

type Stream interface {
	PerformOperation(ctx context.Context, input PerformOperationInput) error
	ReadMessages(ctx context.Context, apply ApplyFunc) error
}

type PerformOperationInput struct {
	RequestID string        `json:"req_id,omitempty"`
	Operation OperationType `json:"op"`
	Arguments []string      `json:"args"`
}

type OperationType string

const (
	OperationTypeSubscribe      OperationType = "subscribe"
	OperationTypeUnsubscribe    OperationType = "unsubscribe"
	OperationTypeAuthentication OperationType = "auth"
	OperationTypePing           OperationType = "ping"
)

type ApplyFunc func(ctx context.Context, event []byte)
