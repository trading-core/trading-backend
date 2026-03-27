package broker

import (
	"context"
	"strconv"

	"github.com/kduong/trading-backend/internal/broker/tastytrade"
)

type TastyTradeAdapter struct {
	account *Account
	client  tastytrade.Client
}

type NewTastyTradeAdapterInput struct {
	Account *Account
	Client  tastytrade.Client
}

func NewTastyTradeAdapter(input NewTastyTradeAdapterInput) *TastyTradeAdapter {
	return &TastyTradeAdapter{
		account: input.Account,
		client:  input.Client,
	}
}

func (adapter *TastyTradeAdapter) GetBalanceInfo(ctx context.Context) (output *BalanceInfo, err error) {
	accountID := adapter.account.TastyTrade.ID
	tastyTradeAccountBalance, err := adapter.client.GetAccountBalance(ctx, accountID)
	if err != nil {
		return
	}
	balance, err := strconv.ParseFloat(tastyTradeAccountBalance.Data.EquityBuyingPower, 64)
	if err != nil {
		return
	}
	output = &BalanceInfo{
		Account:  adapter.account,
		Balance:  balance,
		Currency: tastyTradeAccountBalance.Data.Currency,
	}
	return
}
