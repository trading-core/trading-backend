package main

import (
	"context"
	"fmt"
	"net/http"
	"tradingbot/bybit"
	"tradingbot/internal/config"
	"tradingbot/internal/fatal"
)

func main() {
	ctx := context.Background()
	client := NewBybitClient()
	serverTime, err := client.GetServerTime(ctx)
	fatal.OnError(err)
	walletBalance, err := client.GetWalletBalance(ctx, bybit.GetWalletBalanceInput{
		AccountType:        bybit.AccountTypeUnified,
		TimestampUnixMilli: serverTime.UnixMilli,
		Currency:           bybit.CurrencyBitcoin,
	})
	fatal.OnError(err)
	fmt.Println(walletBalance)
	fmt.Println(string(fatal.UnlessMarshal(walletBalance)))
}

func NewBybitClient() bybit.Client {
	bybitURL := config.EnvBaseURLOrFatal("BYBIT")
	return &bybit.ClientHTTP{
		Client:      http.DefaultClient,
		Scheme:      bybitURL.Scheme,
		Host:        bybitURL.Host,
		BybitKey:    config.EnvStringOrFatal("BYBIT_API_KEY"),
		BybitSecret: config.EnvStringOrFatal("BYBIT_API_SECRET"),
	}
}
