package broker

import (
	"context"
)

type AccountClient interface {
	GetBalance(ctx context.Context) (*GetBalanceOutput, error)
}

type GetBalanceOutput struct {
	NetLiquidatingValue float64 `json:"net_liquidating_value"`
	CashBalance         float64 `json:"cash_balance"`
	EquityBuyingPower   float64 `json:"equity_buying_power"`
	Currency            string  `json:"currency"`
}

type AccountClientFactory interface {
	Get(ctx context.Context, account *Account) AccountClient
}
