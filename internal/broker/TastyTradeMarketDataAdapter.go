package broker

import (
	"context"
	"time"

	"github.com/kduong/trading-backend/internal/broker/tastytrade"
	"github.com/kduong/trading-backend/internal/iterator"
)

type TastyTradeMarketDataAdapter struct {
	client tastytrade.Client
}

type NewTastyTradeMarketDataAdapterInput struct {
	Client tastytrade.Client
}

func NewTastyTradeMarketDataAdapter(input NewTastyTradeMarketDataAdapterInput) *TastyTradeMarketDataAdapter {
	return &TastyTradeMarketDataAdapter{
		client: input.Client,
	}
}

func (adapter *TastyTradeMarketDataAdapter) Stream(ctx context.Context, input StreamMarketDataInput) iterator.Iterator[*MarketDataMessage] {
	return &tastyTradeMarketDataIterator{
		iterator: tastytrade.NewDXLinkIterator(ctx, tastytrade.NewDXLinkIteratorInput{
			Client: adapter.client,
			Symbol: input.Symbol,
		}),
	}
}

func (adapter *TastyTradeMarketDataAdapter) GetHistoricalData(ctx context.Context, input GetHistoricaDataInput) iterator.Iterator[*MarketDataMessage] {
	return &tastyTradeHistoricalMarketDataIterator{
		iterator: tastytrade.NewHistoricalDXLinkIterator(ctx, tastytrade.NewHistoricalDXLinkIteratorInput{
			Client:         adapter.client,
			Symbol:         input.Symbol,
			CandleInterval: input.CandleInterval,
			FromTime:       input.FromTime,
		}),
	}
}

type tastyTradeMarketDataIterator struct {
	iterator *tastytrade.DXLinkIterator
	item     *MarketDataMessage
}

func (iterator *tastyTradeMarketDataIterator) Next() bool {
	for iterator.iterator.Next() {
		message := iterator.iterator.MessageEvent()
		iterator.item = convertTastyTradeMessageEvent(message)
		return true
	}
	iterator.item = nil
	return false
}

func (iterator *tastyTradeMarketDataIterator) Item() *MarketDataMessage {
	return iterator.item
}

func (iterator *tastyTradeMarketDataIterator) Err() error {
	return iterator.iterator.Err()
}

type tastyTradeHistoricalMarketDataIterator struct {
	iterator *tastytrade.HistoricalDXLinkIterator
	item     *MarketDataMessage
}

func (iterator *tastyTradeHistoricalMarketDataIterator) Next() bool {
	for iterator.iterator.Next() {
		message := iterator.iterator.MessageEvent()
		iterator.item = convertTastyTradeMessageEvent(message)
		return true
	}
	iterator.item = nil
	return false
}

func (iterator *tastyTradeHistoricalMarketDataIterator) Item() *MarketDataMessage {
	return iterator.item
}

func (iterator *tastyTradeHistoricalMarketDataIterator) Err() error {
	return iterator.iterator.Err()
}

func convertTastyTradeMessageEvent(message *tastytrade.MessageEvent) *MarketDataMessage {
	switch message.Type {
	case tastytrade.MessageEventTypeQuote:
		return &MarketDataMessage{
			Type:   MarketDataTypeQuote,
			Symbol: message.Quote.EventSymbol,
			Quote: &Quote{
				BidPrice: message.Quote.BidPrice,
				AskPrice: message.Quote.AskPrice,
				BidSize:  message.Quote.BidSize,
				AskSize:  message.Quote.AskSize,
			},
			ReceivedAt: eventReceivedAt(message.Quote.EventTime),
		}
	case tastytrade.MessageEventTypeTrade:
		return &MarketDataMessage{
			Type:   MarketDataTypeTrade,
			Symbol: message.Trade.EventSymbol,
			Trade: &Trade{
				Price:     message.Trade.Price,
				DayVolume: message.Trade.DayVolume,
				Size:      message.Trade.Size,
			},
			ReceivedAt: eventReceivedAt(message.Trade.EventTime),
		}
	case tastytrade.MessageEventTypeCandle:
		return &MarketDataMessage{
			Type:   MarketDataTypeCandle,
			Symbol: message.Candle.EventSymbol,
			Candle: &Candle{
				Open:         message.Candle.Open,
				High:         message.Candle.High,
				Low:          message.Candle.Low,
				Close:        message.Candle.Close,
				Volume:       message.Candle.Volume,
				OpenInterest: message.Candle.OpenInterest,
			},
			ReceivedAt: eventReceivedAt(message.Candle.EventTime),
		}
	default:
		return nil
	}
}

func eventReceivedAt(eventTime *time.Time) time.Time {
	if eventTime == nil {
		return time.Now()
	}
	return *eventTime
}
