package bybit

import (
	"context"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type StreamWebsocket struct {
	connectionType ConnectionType
	connection     *websocket.Conn
	bybitKey       string
	bybitSecret    string
}

func (stream *StreamWebsocket) PerformOperation(ctx context.Context, input PerformOperationInput) (err error) {
	err = stream.connection.WriteJSON(input)
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var messageType int
			var message []byte
			messageType, message, err = stream.connection.ReadMessage()
			if err != nil {
				log.Printf("read error: %v", err)
				return
			}

			switch messageType {
			case websocket.TextMessage:
				fmt.Println("Received text:", string(message))
			case websocket.BinaryMessage:
				fmt.Println("Received binary:", message)
			default:
				fmt.Println("Received unknown type")
			}
		}
	}
}
