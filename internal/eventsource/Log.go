package eventsource

type Log interface {
	Channel() string
	Append(data []byte) (event *Event, err error)
	Read(cursor int64, limit int, timeoutMS int64) (events []*Event, nextCursor int64, err error)
}

type Event struct {
	LogID    string `json:"log_id"`
	Sequence int64  `json:"sequence"`
	Data     []byte `json:"data"`
}

type LogFactory interface {
	Create(channel string) (log Log, err error)
}
