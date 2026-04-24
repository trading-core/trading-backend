package broker

import (
	"context"
)

type AccountClient interface {
	GetBalance(ctx context.Context) (*GetBalanceOutput, error)
	GetEquityPosition(ctx context.Context, symbol string) (*GetEquityPositionOutput, error)
	PlaceOrder(ctx context.Context, input PlaceOrderInput) (*PlaceOrderOutput, error)
	HasPendingOrder(ctx context.Context, symbol string) (bool, error)
	GetTransactions(ctx context.Context, input GetTransactionsInput) (*GetTransactionsOutput, error)
}

type GetTransactionsInput struct {
	From string
	To   string
}

type GetTransactionsOutput struct {
	Transactions []Transaction
}

type Transaction struct {
	ID          string
	Symbol      string
	Action      OrderAction
	Quantity    float64
	Price       float64
	Value       float64
	Fees        float64
	RealizedPnL float64
	ExecutedAt  string
	Type        string
}

type GetEquityPositionOutput struct {
	Quantity float64
}

type GetBalanceOutput struct {
	NetLiquidatingValue float64 `json:"net_liquidating_value"`
	CashBalance         float64 `json:"cash_balance"`
	EquityBuyingPower   float64 `json:"equity_buying_power"`
	Currency            string  `json:"currency"`
}

type OrderAction string

const (
	OrderActionBuy  OrderAction = "buy"
	OrderActionSell OrderAction = "sell"
)

type PlaceOrderInput struct {
	Symbol   string
	Action   OrderAction
	Quantity float64
}

type PlaceOrderOutput struct {
	OrderID int
}

type AccountClientFactory interface {
	Get(ctx context.Context, account *Account) AccountClient
}
