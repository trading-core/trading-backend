package alpaca

import (
	"context"
	"time"

	"github.com/kduong/trading-backend/internal/config"
)

type Client interface {
	GetActiveStocks(ctx context.Context, input GetActiveStocksInput) (output *GetActiveStocksOutput, err error)
}

type RankBy string

const (
	RankByVolume     RankBy = "volume"
	RankByTradeCount RankBy = "trades"
)

type GetActiveStocksInput struct {
	RankBy RankBy
	Limit  int
}

type GetActiveStocksOutput struct {
	LastUpdated string        `json:"last_updated"`
	MostActives []ActiveStock `json:"most_actives"`
}

type ActiveStock struct {
	Symbol     string `json:"symbol"`
	TradeCount int    `json:"trade_count"`
	Volume     int64  `json:"volume"`
}

func FromEnv() Client {
	implementation := config.EnvString("ALPACA_CLIENT_IMPLEMENTATION", "HTTP")
	switch implementation {
	case "HTTP":
		return NewHTTPClient(NewHTTPClientInput{
			Timeout:   config.EnvDuration("ALPACA_CLIENT_IMPLEMENTATION", 20*time.Second),
			BaseURL:   config.EnvURLOrFatal("ALPACA_API"),
			KeyID:     config.EnvStringOrFatal("ALPACA_API_KEY_ID"),
			SecretKey: config.EnvStringOrFatal("ALPACA_API_SECRET_KEY"),
		})
	default:
		panic("unknown alpaca client implementation: " + implementation)
	}
}
