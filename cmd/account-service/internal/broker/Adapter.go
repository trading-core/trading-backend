package broker

import (
	"context"
)

type Adapter interface {
	GetBalanceInfo(ctx context.Context) (*BalanceInfo, error)
}

type BalanceInfo struct {
	Broker   string  `json:"broker"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}
