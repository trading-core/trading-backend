package bybit

type StreamFactory interface {
	Connect(connectionType ConnectionType) (Stream, error)
}

type ConnectionType int

const (
	LinearPublic ConnectionType = iota
	SpotPublic
	Private
	InversePublic
)
