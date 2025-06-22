package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kduong/tradingbot/bybit"
	"github.com/kduong/tradingbot/internal/config"
	"github.com/kduong/tradingbot/internal/fatal"
	uuid "github.com/satori/go.uuid"
)

func main() {
	ctx := context.Background()
	client := NewBybitClient()
	streamFactory := NewBybitStreamFactory()
	stream, err := streamFactory.Connect(bybit.LinearPublic)
	fatal.OnError(err)

	stream.PerformOperation(ctx, bybit.PerformOperationInput{
		RequestID: uuid.NewV4().String(),
		Operation: bybit.OperationTypeSubscribe,
		Arguments: []string{"publicTrade.BTCUSDT"},
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
