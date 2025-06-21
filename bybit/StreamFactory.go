package bybit

type StreamFactory interface {
	Connect(streamType StreamType) (Stream, error)
}

type StreamType int

const (
	LinearPublic StreamType = iota
	SpotPublic
	Private
	InversePublic
)
