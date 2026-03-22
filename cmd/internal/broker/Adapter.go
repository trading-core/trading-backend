package broker

import (
	"context"

	"github.com/kduong/trading-backend/cmd/internal/account"
)

type Adapter interface {
	GetBalanceInfo(ctx context.Context) (*BalanceInfo, error)
}

type BalanceInfo struct {
	AccountBroker account.BrokerType `json:"account_broker"`
	Balance       float64            `json:"balance"`
	Currency      string             `json:"currency"`
}
