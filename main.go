package main

import (
	"context"
	"net/http"

	"github.com/kduong/tradingbot/bybit"
	"github.com/kduong/tradingbot/internal/config"
	"github.com/kduong/tradingbot/internal/fatal"
	"github.com/kduong/tradingbot/streamsync"
	uuid "github.com/satori/go.uuid"
)

func main() {
	ctx := context.Background()
	streamFactory := NewBybitStreamFactory()
	linearPublicStream, err := streamFactory.Connect(bybit.LinearPublic)
	fatal.OnError(err)
	streamSyncActor := &streamsync.Actor{
		Client: NewBybitClient(),
	}
	err = linearPublicStream.PerformOperation(ctx, bybit.PerformOperationInput{
		RequestID: uuid.NewV4().String(),
		Operation: bybit.OperationTypeSubscribe,
		Arguments: []string{
			"publicTrade.BTCUSDT",
			"publicTrade.ETHUSDT",
			"publicTrade.SOLUSDT",
		},
	})
	fatal.OnError(err)
	go func() {
		err = linearPublicStream.ReadMessages(ctx, streamSyncActor.ApplyMessage)
		fatal.OnError(err)
	}()

	// serverTime, err := client.GetServerTime(ctx)
	// fatal.OnError(err)
	// walletBalance, err := client.GetWalletBalance(ctx, bybit.GetWalletBalanceInput{
	// 	AccountType:        bybit.AccountTypeUnified,
	// 	TimestampUnixMilli: serverTime.UnixMilli,
	// 	Currency:           bybit.CurrencyBitcoin,
	// })
	// fatal.OnError(err)
	// fmt.Println(walletBalance)
	// fmt.Println(string(fatal.UnlessMarshal(walletBalance)))
	select {}
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
