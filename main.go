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
	streamFactory := NewBybitStreamFactory()
	stream, err := streamFactory.Connect(bybit.LinearPublic)
	fatal.OnError(err)

	var arguments []bybit.SubscribeInputArgument
	arguments = append(arguments, bybit.SubscribeInputArgument{
		Topic:  "publicTrade",
		Symbol: "BTCUSDT",
	})
	stream.Subscribe(ctx, bybit.SubscribeInput{
		RequestID: nil,
		Arguments: arguments,
	})

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
	bybitAPIURL := config.EnvBaseURLOrFatal("BYBIT_API")
	return &bybit.ClientHTTP{
		Client:      http.DefaultClient,
		Scheme:      bybitAPIURL.Scheme,
		Host:        bybitAPIURL.Host,
		BybitKey:    config.EnvStringOrFatal("BYBIT_API_KEY"),
		BybitSecret: config.EnvStringOrFatal("BYBIT_API_SECRET"),
	}
}

func NewBybitStreamFactory() bybit.StreamFactory {
	bybitStreamURL := config.EnvBaseURLOrFatal("BYBIT_STREAM")
	return &bybit.StreamWebsocketFactory{
		Scheme:      bybitStreamURL.Scheme,
		Host:        bybitStreamURL.Host,
		BybitKey:    config.EnvStringOrFatal("BYBIT_API_KEY"),
		BybitSecret: config.EnvStringOrFatal("BYBIT_API_SECRET"),
	}
}
