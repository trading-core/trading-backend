package broker

import (
	"context"
	"strconv"

	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type TastyTradeAccountAdapter struct {
	accountID string
	client    tastytrade.Client
}

type NewTastyTradeAccountAdapterInput struct {
	AccountID string
	Client    tastytrade.Client
}

func NewTastyTradeAccountAdapter(input NewTastyTradeAccountAdapterInput) *TastyTradeAccountAdapter {
	return &TastyTradeAccountAdapter{
		accountID: input.AccountID,
		client:    input.Client,
	}
}

func (adapter *TastyTradeAccountAdapter) GetBalance(ctx context.Context) (output *GetBalanceOutput, err error) {
	tastyTradeAccountBalance, err := adapter.client.GetAccountBalance(ctx, adapter.accountID)
	if err != nil {
		return
	}
	balance, err := strconv.ParseFloat(tastyTradeAccountBalance.Data.EquityBuyingPower, 64)
	if err != nil {
		return
	}
	output = &GetBalanceOutput{
		Balance:  balance,
		Currency: tastyTradeAccountBalance.Data.Currency,
	}
	return
}
