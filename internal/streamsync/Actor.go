package streamsync

import (
	"context"
	"fmt"

	"github.com/kduong/trading-backend/internal/bybit"
)

type Actor struct {
	Client bybit.Client
}

func (actor *Actor) ApplyMessage(ctx context.Context, event []byte) {
	// var message TradeMessage
	// fatal.UnlessUnmarshal(event, &message)

	// fmt.Println(message)

	fmt.Println(string(event))
	fmt.Println()
}

type TradeMessage struct {
	Topic string       `json:"topic"`
	Type  string       `json:"type"`
	TS    int64        `json:"ts"`
	Data  []TradeEntry `json:"data"`
}

type TradeEntry struct {
	T             int64  `json:"T"`   // Trade timestamp
	Symbol        string `json:"s"`   // Symbol (e.g., ETHUSDT)
	Side          string `json:"S"`   // "Buy" or "Sell"
	Volume        string `json:"v"`   // Trade size
	Price         string `json:"p"`   // Trade price
	TickDirection string `json:"L"`   // "PlusTick", "MinusTick"
	ID            string `json:"i"`   // Trade ID
	BT            bool   `json:"BT"`  // Block trade flag
	RPI           bool   `json:"RPI"` // Reduce-only flag
}

type OperationMessage struct {
	Success       bool                `json:"success"`
	ReturnMessage string              `json:"retMsg"`
	ConnectionID  string              `json:"conn_id"`
	RequestID     string              `json:"req_id,omitempty"`
	Operation     bybit.OperationType `json:"op"`
}
