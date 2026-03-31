package oauthstatestore

import (
	"sync"
	"time"
)

var _ Store = (*InMemory)(nil)

type InMemory struct {
	mutex   sync.Mutex
	entries map[string]Entry
}

func NewInMemory() *InMemory {
	return &InMemory{
		entries: make(map[string]Entry),
	}
}

func (store *InMemory) Put(token string, entry Entry) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	for key, value := range store.entries {
		if time.Now().After(value.ExpiresAt) {
			delete(store.entries, key)
		}
	}
	store.entries[token] = entry
}

func (store *InMemory) Pop(token string) (Entry, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	entry, ok := store.entries[token]
	if !ok {
		return Entry{}, false
	}
	delete(store.entries, token)
	if time.Now().After(entry.ExpiresAt) {
		return Entry{}, false
	}
	return entry, true
}
