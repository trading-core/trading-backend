package bybit

import (
	"net/http"
	"net/url"
	"tradingbot/internal/fatal"

	"github.com/ansel1/merry"
	"github.com/gorilla/websocket"
)

type StreamWebsocketFactory struct {
	Scheme      string
	Host        string
	BybitKey    string
	BybitSecret string
}

func (factory *StreamWebsocketFactory) Connect(connectionType ConnectionType) (stream Stream, err error) {
	endpoint, ok := getConnectionEndpointByType[connectionType]
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
		connectionType: connectionType,
		connection:     connection,
		bybitKey:       factory.BybitKey,
		bybitSecret:    factory.BybitSecret,
	}
	return
}

var getConnectionEndpointByType = map[ConnectionType]string{
	LinearPublic:  "/v5/public/linear",
	SpotPublic:    "/v5/public/spot",
	Private:       "/v5/private",
	InversePublic: "/v5/public/inverse",
}
