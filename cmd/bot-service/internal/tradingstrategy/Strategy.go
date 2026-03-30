package tradingstrategy

import (
	"context"
	"errors"
	"time"

	"github.com/kduong/trading-backend/cmd/bot-service/internal/botstore"
)

var ErrUnknownStrategyType = errors.New("unknown strategy type")

type Action string

const (
	ActionNone Action = "none"
	ActionBuy  Action = "buy"
	ActionSell Action = "sell"
	ActionExit Action = "exit"
)

type EvaluateInput struct {
	Bot              *botstore.Bot
	Price            float64
	SessionOpenPrice float64
	SessionHighPrice float64
	SessionLowPrice  float64
	CashBalance      float64
	BuyingPower      float64
	PositionQuantity float64
	HasOpenOrder     bool
	Now              time.Time
}

type Decision struct {
	Action   Action
	Reason   string
	Quantity float64
}

type Strategy interface {
	Type() string
	Validate(bot *botstore.Bot) error
	Evaluate(ctx context.Context, input EvaluateInput) (Decision, error)
}
