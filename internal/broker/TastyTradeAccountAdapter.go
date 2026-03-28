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
	data := tastyTradeAccountBalance.Data
	netLiquidatingValue, err := strconv.ParseFloat(data.NetLiquidatingValue, 64)
	if err != nil {
		return
	}
	cashBalance, err := strconv.ParseFloat(data.CashBalance, 64)
	if err != nil {
		return
	}
	equityBuyingPower, err := strconv.ParseFloat(data.EquityBuyingPower, 64)
	if err != nil {
		return
	}
	output = &GetBalanceOutput{
		NetLiquidatingValue: netLiquidatingValue,
		CashBalance:         cashBalance,
		EquityBuyingPower:   equityBuyingPower,
		Currency:            data.Currency,
	}
	return
}
