package tastytrade

import (
	"context"
	"math"
	"time"

	"github.com/gorilla/websocket"
)

type DXLinkIterator struct {
	client        Client
	symbol        string
	messageEventC chan *MessageEvent
	messageEvent  *MessageEvent
	err           error
}

type NewDXLinkIteratorInput struct {
	Client Client
	Symbol string
}

func NewDXLinkIterator(ctx context.Context, input NewDXLinkIteratorInput) *DXLinkIterator {
	iterator := &DXLinkIterator{
		client:        input.Client,
		symbol:        input.Symbol,
		messageEventC: make(chan *MessageEvent, 32),
	}
	go iterator.run(ctx)
	return iterator
}

func (iterator *DXLinkIterator) Next() bool {
	item, ok := <-iterator.messageEventC
	if !ok {
		return false
	}
	iterator.messageEvent = item
	return true
}

func (iterator *DXLinkIterator) MessageEvent() *MessageEvent {
	return iterator.messageEvent
}

func (iterator *DXLinkIterator) Err() error {
	return iterator.err
}

func (iterator *DXLinkIterator) run(ctx context.Context) {
	defer close(iterator.messageEventC)
	apiQuoteToken, err := iterator.client.GetAPIQuoteToken(ctx)
	if err != nil {
		iterator.err = err
		return
	}
	connection, _, err := websocket.DefaultDialer.DialContext(ctx, apiQuoteToken.Data.DXLinkURL, nil)
	if err != nil {
		iterator.err = err
		return
	}
	defer connection.Close()
	go func() {
		<-ctx.Done()
	}()
	err = iterator.initiateDXLinkConnection(connection)
	if err != nil {
		iterator.err = err
		return
	}
	err = iterator.authorize(ctx, connection, apiQuoteToken.Data.Token)
	if err != nil {
		iterator.err = err
		return
	}
	err = iterator.openFeed(connection)
	if err != nil {
		iterator.err = err
		return
	}
	go iterator.keepConnectionAlive(ctx, connection)
	for {
		var rawMessage map[string]any
		err = connection.ReadJSON(&rawMessage)
		if err != nil {
			if ctx.Err() == nil {
				iterator.err = err
			}
			return
		}
		messages, ok := parseRawMessage(rawMessage)
		if !ok {
			continue
		}
		for _, message := range messages {
			select {
			case <-ctx.Done():
				return
			case iterator.messageEventC <- message:
			}
		}
	}
}

func (iterator *DXLinkIterator) initiateDXLinkConnection(connection *websocket.Conn) (err error) {
	return connection.WriteJSON(map[string]any{
		"type":                   "SETUP",
		"channel":                0,
		"version":                "0.1-DXF-GO/1.0",
		"keepaliveTimeout":       60,
		"acceptKeepaliveTimeout": 60,
	})
}

func (iterator *DXLinkIterator) authorize(ctx context.Context, connection *websocket.Conn, token string) (err error) {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var message map[string]any
		err = connection.ReadJSON(&message)
		if err != nil {
			return
		}
		if message["type"] != "AUTH_STATE" {
			continue
		}
		switch message["state"] {
		case "AUTHORIZED":
			return nil
		case "UNAUTHORIZED":
			payload := map[string]any{
				"type":    "AUTH",
				"channel": 0,
				"token":   token,
			}
			err = connection.WriteJSON(payload)
			if err != nil {
				return
			}
			continue
		}
	}
}

func (iterator *DXLinkIterator) openFeed(connection *websocket.Conn) (err error) {
	err = connection.WriteJSON(map[string]any{
		"type":    "CHANNEL_REQUEST",
		"channel": 3,
		"service": "FEED",
		"parameters": map[string]any{
			"contract": "AUTO",
		},
	})
	if err != nil {
		return
	}
	err = connection.WriteJSON(map[string]any{
		"type":                      "FEED_SETUP",
		"channel":                   3,
		"acceptedAggregationPeriod": 0.1,
		"acceptedDataFormat":        "COMPACT",
		"acceptEventFields": map[string]any{
			"Quote": []string{"eventType", "eventSymbol", "bidPrice", "askPrice", "bidSize", "askSize"},
			"Trade": []string{"eventType", "eventSymbol", "price", "dayVolume", "size"},
		},
	})
	if err != nil {
		return
	}
	return connection.WriteJSON(map[string]any{
		"type":    "FEED_SUBSCRIPTION",
		"channel": 3,
		"reset":   true,
		"add": []map[string]any{
			{"type": "Quote", "symbol": iterator.symbol},
			{"type": "Trade", "symbol": iterator.symbol},
		},
	})
}

func (iterator *DXLinkIterator) keepConnectionAlive(ctx context.Context, connection *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := connection.WriteJSON(map[string]any{
				"type":    "KEEPALIVE",
				"channel": 0,
			})
			if err != nil {
				if ctx.Err() == nil && iterator.err == nil {
					iterator.err = err
				}
				return
			}
		}
	}
}

func parseRawMessage(rawMessage map[string]any) ([]*MessageEvent, bool) {
	messageType := rawMessage["type"].(string)
	if messageType != "FEED_DATA" {
		return nil, false
	}
	data, ok := rawMessage["data"].([]any)
	if !ok {
		return nil, false
	}
	var messages []*MessageEvent
	for _, entry := range data {
		event := entry.(map[string]any)
		eventType := MessageEventType(event["eventType"].(string))
		eventSymbol := event["eventSymbol"].(string)
		switch eventType {
		case MessageEventTypeQuote:
			messages = append(messages, &MessageEvent{
				Type: MessageEventTypeQuote,
				Quote: &MessageEventQuote{
					EventSymbol: eventSymbol,
					BidPrice:    numberValue(event["bidPrice"]),
					AskPrice:    numberValue(event["askPrice"]),
					BidSize:     numberValue(event["bidSize"]),
					AskSize:     numberValue(event["askSize"]),
				},
			})
		case MessageEventTypeTrade:
			messages = append(messages, &MessageEvent{
				Type: MessageEventTypeTrade,
				Trade: &MessageEventTrade{
					EventSymbol: eventSymbol,
					Price:       numberValue(event["price"]),
					DayVolume:   optionalNumberValue(event["dayVolume"]),
					Size:        optionalNumberValue(event["size"]),
				},
			})
		}
	}
	return messages, true
}

func optionalNumberValue(value any) *float64 {
	v := numberValue(value)
	if math.IsNaN(v) {
		return nil
	}
	return &v
}

func numberValue(value any) float64 {
	switch typedValue := value.(type) {
	case float64:
		return typedValue
	default:
		return math.NaN()
	}
}
