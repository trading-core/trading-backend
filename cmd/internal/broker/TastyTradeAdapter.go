package broker

import "context"

type TastyTradeAdapter struct {
}

func NewTastyTradeAdapter() *TastyTradeAdapter {
	return &TastyTradeAdapter{}
}

func (adapter *TastyTradeAdapter) GetBalanceInfo(ctx context.Context) (output *BalanceInfo, err error) {
	return
}
