package botsync

import "github.com/kduong/trading-backend/cmd/bot-service/internal/tradingstrategy"

type Actor struct {
	TradingStrategy tradingstrategy.Strategy
}
