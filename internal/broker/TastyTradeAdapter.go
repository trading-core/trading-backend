package broker

import (
	"context"
	"strconv"

	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type TastyTradeAdapter struct {
	accountID string
	client    tastytrade.Client
}

type NewTastyTradeAdapterInput struct {
	AccountID string
	Client    tastytrade.Client
}

func NewTastyTradeAdapter(input NewTastyTradeAdapterInput) *TastyTradeAdapter {
	return &TastyTradeAdapter{
		accountID: input.AccountID,
		client:    input.Client,
	}
}

func (adapter *TastyTradeAdapter) GetBalance(ctx context.Context) (output *GetBalanceOutput, err error) {
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
