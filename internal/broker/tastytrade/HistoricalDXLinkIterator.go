package tastytrade

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// HistoricalDXLinkIterator streams Candle events from a fromTime and then continues with live candle updates.
type HistoricalDXLinkIterator struct {
	client        Client
	symbol        string
	fromTime      time.Time
	messageEventC chan *MessageEvent
	messageEvent  *MessageEvent
	err           error
}

type NewHistoricalDXLinkIteratorInput struct {
	Client         Client
	Symbol         string
	CandleInterval string
	FromTime       time.Time
}

func NewHistoricalDXLinkIterator(ctx context.Context, input NewHistoricalDXLinkIteratorInput) *HistoricalDXLinkIterator {
	iterator := &HistoricalDXLinkIterator{
		client:        input.Client,
		symbol:        candleSymbol(input.Symbol, input.CandleInterval),
		fromTime:      input.FromTime,
		messageEventC: make(chan *MessageEvent, 32),
	}
	go iterator.run(ctx)
	return iterator
}

func (iterator *HistoricalDXLinkIterator) Next() bool {
	item, ok := <-iterator.messageEventC
	if !ok {
		return false
	}
	iterator.messageEvent = item
	return true
}

func (iterator *HistoricalDXLinkIterator) MessageEvent() *MessageEvent {
	return iterator.messageEvent
}

func (iterator *HistoricalDXLinkIterator) Err() error {
	return iterator.err
}

func (iterator *HistoricalDXLinkIterator) run(ctx context.Context) {
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
	if err = iterator.initiateDXLinkConnection(connection); err != nil {
		iterator.err = err
		return
	}
	if err = iterator.authorize(ctx, connection, apiQuoteToken.Data.Token); err != nil {
		iterator.err = err
		return
	}
	if err = iterator.openHistoricalFeed(connection); err != nil {
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
		messages, ok := parseRawHistoricalMessage(rawMessage)
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

func (iterator *HistoricalDXLinkIterator) initiateDXLinkConnection(connection *websocket.Conn) (err error) {
	return connection.WriteJSON(map[string]any{
		"type":                   "SETUP",
		"channel":                0,
		"version":                "0.1-DXF-GO/1.0",
		"keepaliveTimeout":       60,
		"acceptKeepaliveTimeout": 60,
	})
}

func (iterator *HistoricalDXLinkIterator) authorize(ctx context.Context, connection *websocket.Conn, token string) (err error) {
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

func (iterator *HistoricalDXLinkIterator) openHistoricalFeed(connection *websocket.Conn) (err error) {
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
			"Candle": []string{"eventType", "eventSymbol", "open", "high", "low", "close", "volume", "openInterest", "time"},
		},
	})
	if err != nil {
		return
	}
	fromTime := iterator.fromTime
	if fromTime.IsZero() {
		fromTime = time.Now().Add(-24 * time.Hour)
	}
	return connection.WriteJSON(map[string]any{
		"type":    "FEED_SUBSCRIPTION",
		"channel": 3,
		"reset":   true,
		"add": []map[string]any{
			{
				"type":     "Candle",
				"symbol":   iterator.symbol,
				"fromTime": fromTime.UnixMilli(),
			},
		},
	})
}

func (iterator *HistoricalDXLinkIterator) keepConnectionAlive(ctx context.Context, connection *websocket.Conn) {
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

func parseRawHistoricalMessage(rawMessage map[string]any) ([]*MessageEvent, bool) {
	messageType, ok := rawMessage["type"].(string)
	if !ok || messageType != "FEED_DATA" {
		return nil, false
	}
	data, ok := rawMessage["data"].([]any)
	if !ok {
		return nil, false
	}
	var messages []*MessageEvent
	for _, entry := range data {
		event, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		eventType := MessageEventType(stringValue(event["eventType"]))
		if eventType != MessageEventTypeCandle {
			continue
		}
		messages = append(messages, &MessageEvent{
			Type: MessageEventTypeCandle,
			Candle: &MessageEventCandle{
				EventSymbol:  stringValue(event["eventSymbol"]),
				Open:         numberValue(event["open"]),
				High:         numberValue(event["high"]),
				Low:          numberValue(event["low"]),
				Close:        numberValue(event["close"]),
				Volume:       optionalNumberValue(event["volume"]),
				OpenInterest: optionalNumberValue(event["openInterest"]),
				EventTime:    optionalTimestampValue(event["time"]),
			},
		})
	}
	if len(messages) == 0 {
		return nil, false
	}
	return messages, true
}

func stringValue(value any) string {
	switch typedValue := value.(type) {
	case string:
		return typedValue
	default:
		return ""
	}
}

func candleSymbol(symbol string, interval string) string {
	base := strings.TrimSpace(symbol)
	if base == "" {
		return ""
	}
	if strings.Contains(base, "{=") {
		return base
	}
	cleanInterval := strings.TrimSpace(interval)
	if cleanInterval == "" {
		cleanInterval = "1m"
	}
	return fmt.Sprintf("%s{=%s}", base, cleanInterval)
}
