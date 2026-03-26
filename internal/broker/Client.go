package broker

import (
	"context"
)

type Client interface {
	GetBalanceInfo(ctx context.Context) (*BalanceInfo, error)
}

type BalanceInfo struct {
	Account  *Account `json:"account"`
	Balance  float64  `json:"balance"`
	Currency string   `json:"currency"`
}
