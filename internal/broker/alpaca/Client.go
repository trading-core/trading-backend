package alpaca

import (
	"context"
	"time"

	"github.com/kduong/trading-backend/internal/config"
)

type Client interface {
	GetActiveStocks(ctx context.Context, input GetActiveStocksInput) (output *GetActiveStocksOutput, err error)
	GetTopStockMovers(ctx context.Context, input GetTopStockMoversInput) (output *GetTopStockMoversOutput, err error)
	GetStockNews(ctx context.Context, input GetStockNewsInput) (output *GetStockNewsOutput, err error)
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

type GetTopStockMoversInput struct {
	Limit int
}

type GetTopStockMoversOutput struct {
	LastUpdated string       `json:"last_updated"`
	Gainers     []MoverStock `json:"gainers"`
	Losers      []MoverStock `json:"losers"`
}

type MoverStock struct {
	Change        float32 `json:"change"`
	PercentChange float32 `json:"percent_change"`
	Price         float32 `json:"price"`
	Symbol        string  `json:"symbol"`
}

type GetStockNewsInput struct {
	PageToken string
	Symbols   []string
	Limit     int
}

type GetStockNewsOutput struct {
	LastUpdated   string      `json:"last_updated"`
	News          []StockNews `json:"news"`
	NextPageToken string      `json:"next_page_token,omitempty"`
}

type StockNews struct {
	Author    string       `json:"author"`
	Content   string       `json:"content"`
	CreatedAt string       `json:"created_at"`
	Headline  string       `json:"headline"`
	ID        int64        `json:"id"`
	Images    []StockImage `json:"images"`
	Source    string       `json:"source"`
	Summary   string       `json:"summary"`
	Symbols   []string     `json:"symbols"`
	UpdatedAt string       `json:"updated_at"`
	URL       string       `json:"url"`
}

type StockImage struct {
	Size string `json:"size"`
	URL  string `json:"url"`
}

func ClientFromEnv() Client {
	implementation := config.EnvString("ALPACA_CLIENT_IMPLEMENTATION", "HTTP")
	switch implementation {
	case "HTTP":
		return NewHTTPClient(NewHTTPClientInput{
			Timeout:   config.EnvDuration("ALPACA_CLIENT_IMPLEMENTATION", 20*time.Second),
			BaseURL:   config.EnvURLOrFatal("ALPACA_API"),
			KeyID:     config.EnvStringOrFatal("ALPACA_API_KEY"),
			SecretKey: config.EnvStringOrFatal("ALPACA_API_SECRET"),
		})
	default:
		panic("unknown alpaca client implementation: " + implementation)
	}
}
