package bybit

import (
	"context"

	"github.com/gorilla/websocket"
)

type StreamWebsocket struct {
	connection *websocket.Conn
}

func (stream *StreamWebsocket) Subscribe(ctx context.Context) error {
	return nil
}
