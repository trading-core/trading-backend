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

type OperationType string

const (
	OperationTypeSubscribe      OperationType = "subscribe"
	OperationTypeUnsubscribe    OperationType = "unsubscribe"
	OperationTypeAuthentication OperationType = "auth"
	OperationTypePing           OperationType = "ping"
)

type WebsocketRequest struct {
	RequestID *string       `json:"req_id,omitempty"`
	Operation OperationType `json:"op"`
	Arguments []string      `json:"args"`
}

func (stream *StreamWebsocket) Subscribe(ctx context.Context, input SubscribeInput) (err error) {
	request := stream.buildWebsocketRequest(input)
	err = stream.connection.WriteJSON(request)
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

func (stream *StreamWebsocket) buildWebsocketRequest(input SubscribeInput) WebsocketRequest {
	var arguments []string
	for _, argument := range input.Arguments {
		arguments = append(arguments, argument.String())
	}
	request := WebsocketRequest{
		RequestID: input.RequestID,
		Operation: OperationTypeSubscribe,
		Arguments: arguments,
	}
	return request
}
