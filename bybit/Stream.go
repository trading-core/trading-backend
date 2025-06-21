package bybit

import (
	"context"
	"fmt"
)

type Stream interface {
	Subscribe(ctx context.Context, input SubscribeInput) error
}

type SubscribeInput struct {
	RequestID *string
	Arguments []SubscribeInputArgument
}

type SubscribeInputArgument struct {
	Topic    string  // e.g., "publicTrade", "kline", "orderbook.1"
	Interval *string // Optional, for kline or depth
	Symbol   string  // e.g., "BTCUSDT"
}

func (input SubscribeInputArgument) String() string {
	if input.Interval != nil {
		return fmt.Sprintf("%s.%s.%s", input.Topic, *input.Interval, input.Symbol)
	}
	return fmt.Sprintf("%s.%s", input.Topic, input.Symbol)
}
