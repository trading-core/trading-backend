package tastytrade

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gorilla/websocket"
)

type DXLinkIterator struct {
	client   Client
	symbol   string
	messageC chan Message
	message  Message
	err      error
}

type NewDXLinkIteratorInput struct {
	Client Client
	Symbol string
}

func NewDXLinkIterator(ctx context.Context, input NewDXLinkIteratorInput) *DXLinkIterator {
	iterator := &DXLinkIterator{
		client:   input.Client,
		symbol:   input.Symbol,
		messageC: make(chan Message, 32),
	}
	go iterator.run(ctx)
	return iterator
}

func (iterator *DXLinkIterator) run(ctx context.Context) {
	defer close(iterator.messageC)
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
	keepAliveTicker := time.NewTicker(30 * time.Second)
	defer keepAliveTicker.Stop()
	go iterator.sendKeepAlives(ctx, connection, keepAliveTicker)
	for {
		var rawMessage map[string]any
		err = connection.ReadJSON(&rawMessage)
		if err != nil {
			if ctx.Err() == nil {
				iterator.err = err
			}
			return
		}
		messages, err := parseRawMessage(rawMessage)
		if err != nil {
			continue
		}
		fmt.Println("how many message? ", len(messages))
		for _, message := range messages {
			select {
			case <-ctx.Done():
				return
			case iterator.messageC <- message:
			}
		}
	}
}

func (iterator *DXLinkIterator) initiateDXLinkConnection(connection *websocket.Conn) (err error) {
	return connection.WriteJSON(MessageSetup{
		MessageBase: MessageBase{
			Type:    MessageTypeSetup,
			Channel: 0,
		},
		Version:                "0.1-DXF-GO/1.0",
		KeepaliveTimeout:       60,
		AcceptKeepaliveTimeout: 60,
	})
}

func (iterator *DXLinkIterator) authorize(ctx context.Context, connection *websocket.Conn, token string) (err error) {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var message MessageAuthState
		err = connection.ReadJSON(&message)
		if err != nil {
			return
		}
		if message.Type != MessageTypeAuthState {
			continue
		}
		switch message.State {
		case AuthStateUnauthorized:
			payload := MessageAuth{
				MessageBase: MessageBase{
					Type:    MessageTypeAuth,
					Channel: 0,
				},
				Token: token,
			}
			if err := connection.WriteJSON(payload); err != nil {
				return err
			}
			continue
		case AuthStateAuthorized:
			return nil
		}
	}
}

func (iterator *DXLinkIterator) Next() bool {
	item, ok := <-iterator.messageC
	if !ok {
		return false
	}
	iterator.message = item
	return true
}

func (iterator *DXLinkIterator) Message() Message {
	return iterator.message
}

func (iterator *DXLinkIterator) Err() error {
	return iterator.err
}

func (iterator *DXLinkIterator) openFeed(connection *websocket.Conn) (err error) {
	err = connection.WriteJSON(MessageChannelRequest{
		MessageBase: MessageBase{
			Type:    MessageTypeChannelRequest,
			Channel: 3,
		},
		Service: "FEED",
		Parameters: map[string]any{
			"contract": "AUTO",
		},
	})
	if err != nil {
		return
	}
	err = connection.WriteJSON(MessageFeedSetup{
		MessageBase: MessageBase{
			Type:    MessageTypeFeedSetup,
			Channel: 3,
		},
		AcceptedAggregationPeriod: 0.1,
		AcceptedDataFormat:        "COMPACT",
		AcceptEventFields: map[string]any{
			"Quote": []string{"eventType", "eventSymbol", "bidPrice", "askPrice", "bidSize", "askSize"},
			"Trade": []string{"eventType", "eventSymbol", "price", "dayVolume", "size"},
		},
	})
	if err != nil {
		return
	}
	return connection.WriteJSON(MessageFeedSubscription{
		MessageBase: MessageBase{
			Type:    MessageTypeFeedSubscription,
			Channel: 3,
		},
		Reset: true,
		Add: []map[string]any{
			{"type": "Quote", "symbol": iterator.symbol},
			{"type": "Trade", "symbol": iterator.symbol},
		},
	})
}

func (iterator *DXLinkIterator) sendKeepAlives(ctx context.Context, connection *websocket.Conn, ticker *time.Ticker) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := connection.WriteJSON(MessageKeepAlive{
				MessageBase: MessageBase{
					Type:    MessageTypeKeepAlive,
					Channel: 0,
				},
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

func parseRawMessage(rawMessage map[string]any) (messages []Message, err error) {
	messageType := rawMessage["type"].(string)
	if messageType != "FEED_DATA" {
		return nil, fmt.Errorf("unsupported message type: %s", messageType)
	}
	data, ok := rawMessage["data"].([]any)
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("invalid dxlink feed data payload: missing or empty data array")
	}
	channel := int(numberValue(rawMessage["channel"]))
	for _, entry := range data {
		eventMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		eventType, ok := eventMap["eventType"].(string)
		if !ok {
			continue
		}
		eventSymbol, ok := eventMap["eventSymbol"].(string)
		if !ok {
			continue
		}
		switch eventType {
		case "Quote":
			messages = append(messages, &MessageQuote{
				MessageBase: MessageBase{
					Type:    MessageTypeFeedData,
					Channel: channel,
				},
				EventType:   eventType,
				EventSymbol: eventSymbol,
				BidPrice:    numberValue(eventMap["bidPrice"]),
				AskPrice:    numberValue(eventMap["askPrice"]),
				BidSize:     numberValue(eventMap["bidSize"]),
				AskSize:     numberValue(eventMap["askSize"]),
			})

		case "Trade":
			messages = append(messages, &MessageTrade{
				MessageBase: MessageBase{
					Type:    MessageTypeFeedData,
					Channel: channel,
				},
				EventType:   eventType,
				EventSymbol: eventSymbol,
				Price:       numberValue(eventMap["price"]),
				DayVolume:   numberValue(eventMap["dayVolume"]),
				Size:        numberValue(eventMap["size"]),
			})
		}
	}
	return
}

func numberValue(value any) float64 {
	switch typedValue := value.(type) {
	case float64:
		return typedValue
	case int:
		return float64(typedValue)
	case int64:
		return float64(typedValue)
	case string:
		if typedValue == "NaN" {
			return math.NaN()
		}
		var parsed float64
		_, err := fmt.Sscanf(typedValue, "%f", &parsed)
		if err == nil {
			return parsed
		}
	}
	return 0
}
