package broker

import (
	"context"

	"github.com/kduong/trading-backend/cmd/internal/account"
)

type MockTestAdapter struct {
	balance float64
}

func NewMockTestAdapter() *MockTestAdapter {
	return &MockTestAdapter{
		balance: 100000,
	}
}

func (adapter *MockTestAdapter) GetBalanceInfo(ctx context.Context) (output *BalanceInfo, err error) {
	output = &BalanceInfo{
		AccountBroker: account.BrokerTypeMockTest,
		Balance:       adapter.balance,
		Currency:      "USD",
	}
	return
}
