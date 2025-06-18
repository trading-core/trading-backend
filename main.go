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
		Coin:               "BTC",
	})
	fatal.OnError(err)
	fmt.Println(walletBalance)
}

func NewBybitClient() bybit.Client {
	bybitURL := config.EnvBaseURLOrFatal("BYBIT")
	return &bybit.ClientHTTP{
		Client:      http.DefaultClient,
		Scheme:      bybitURL.Scheme,
		Host:        bybitURL.Host,
		ByBitKey:    config.EnvStringOrFatal("BYBIT_API_KEY"),
		ByBitSecret: config.EnvStringOrFatal("BYBIT_API_SECRET"),
	}
}
