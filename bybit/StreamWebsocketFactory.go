package bybit

import (
	"net/http"
	"net/url"
	"tradingbot/internal/fatal"

	"github.com/ansel1/merry"
	"github.com/gorilla/websocket"
)

type StreamWebsocketFactory struct {
	Scheme string
	Host   string
}

func (factory *StreamWebsocketFactory) Connect(streamType StreamType) (stream Stream, err error) {
	endpoint, ok := getStreamEndpointByType[streamType]
	if !ok {
		err = merry.Errorf("stream connection type (%s) not found").WithHTTPCode(http.StatusBadRequest)
		return
	}
	target := url.URL{
		Scheme: factory.Scheme,
		Host:   factory.Host,
		Path:   endpoint,
	}
	connection, response, err := websocket.DefaultDialer.Dial(target.String(), nil)
	fatal.OnError(err)
	fatal.Unless(response.StatusCode == http.StatusSwitchingProtocols)
	stream = &StreamWebsocket{
		connection: connection,
	}
	return
}

var getStreamEndpointByType = map[StreamType]string{
	LinearPublic:  "/v5/public/linear",
	SpotPublic:    "/v5/public/spot",
	Private:       "/v5/private",
	InversePublic: "/v5/public/inverse",
}
