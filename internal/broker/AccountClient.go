package broker

import (
	"context"
)

type AccountClient interface {
	GetBalance(ctx context.Context) (*GetBalanceOutput, error)
}

type GetBalanceOutput struct {
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}
