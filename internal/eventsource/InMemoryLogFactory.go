package eventsource

import "sync"

type InMemoryLogFactory struct {
	mutex sync.Mutex
	logs  map[string]Log
}

func NewInMemoryLogFactory() *InMemoryLogFactory {
	return &InMemoryLogFactory{
		logs: make(map[string]Log),
	}
}

func (factory *InMemoryLogFactory) Close() error {
	return nil
}

func (factory *InMemoryLogFactory) Create(channel string) (log Log, err error) {
	factory.mutex.Lock()
	defer factory.mutex.Unlock()
	log, ok := factory.logs[channel]
	if !ok {
		log = NewInMemoryLog(channel)
		factory.logs[channel] = log
	}
	return
}
