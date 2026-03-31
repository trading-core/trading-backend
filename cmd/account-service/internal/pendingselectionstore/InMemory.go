package pendingselectionstore

import (
	"sync"
	"time"
)

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
	for k, v := range store.entries {
		if time.Now().After(v.ExpiresAt) {
			delete(store.entries, k)
		}
	}
	store.entries[token] = entry
}

func (store *InMemory) Delete(token string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	delete(store.entries, token)
}

func (store *InMemory) Get(token string) (Entry, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	entry, ok := store.entries[token]
	if !ok {
		return Entry{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(store.entries, token)
		return Entry{}, false
	}
	return entry, true
}
