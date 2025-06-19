package bybit

import "context"

type Client interface {
	GetServerTime(ctx context.Context) (output *ServerTime, err error)
	GetWalletBalance(ctx context.Context, input GetWalletBalanceInput) (output *WalletBalance, err error)
}

type GetWalletBalanceInput struct {
	AccountType        AccountType
	TimestampUnixMilli int64
	Currency           Currency
}
