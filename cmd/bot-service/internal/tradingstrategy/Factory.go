package tradingstrategy

import (
	"fmt"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
)

func New(bot *botstore.Bot) (Strategy, error) {
	if bot == nil {
		return nil, fmt.Errorf("strategy bot config is required")
	}

	var strategy Strategy
	switch bot.StrategyTradeType {
	case "momentum_breakout":
		strategy = MomentumBreakout{}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownStrategyType, bot.StrategyTradeType)
	}

	if err := strategy.Validate(bot); err != nil {
		return nil, err
	}
	return strategy, nil
}
