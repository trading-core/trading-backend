package streamsync

import "github.com/kduong/tradingbot/bybit"

type ParentActor struct {
	Client        bybit.Client
	StreamFactory bybit.StreamFactory
}

func (actor *ParentActor) StartStream()
