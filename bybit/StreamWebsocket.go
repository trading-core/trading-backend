package bybit

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/kduong/tradingbot/internal/logger"
)

type StreamWebsocket struct {
	connectionType ConnectionType
	connection     *websocket.Conn
	bybitKey       string
	bybitSecret    string
}

func (stream *StreamWebsocket) ReadMessages(ctx context.Context, apply ApplyMessageFunc) (err error) {
	var messageType int
	var message []byte
	for {
		select {
		case <-ctx.Done():
			return
		default:
			messageType, message, err = stream.connection.ReadMessage()
			if err != nil {
				return
			}
			switch messageType {
			case websocket.TextMessage:
				apply(ctx, message)
			case websocket.PingMessage:
				stream.connection.WriteMessage(websocket.PongMessage, nil)
			case websocket.PongMessage:
				logger.Notice("Received pong")
			}
		}
	}
}

func (stream *StreamWebsocket) PerformOperation(ctx context.Context, input PerformOperationInput) (err error) {
	return stream.connection.WriteJSON(input)
}
