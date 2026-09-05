package chronicle

import (
	"maps"
	"sync"
)

// entry tells us where the latest value for a key is.
type entry struct {
	fileID    int64
	offset    int64
	size      int32
	timestamp int64
	tombstone bool
}

// keydir is the in-memory map from key to its latest entry.
type keydir struct {
	mu      sync.RWMutex
	entries map[string]entry
}

func newKeydir() *keydir {
	return &keydir{entries: make(map[string]entry)}
}

func (k *keydir) put(key string, e entry) {
	k.mu.Lock()
	k.entries[key] = e
	k.mu.Unlock()
}

func (k *keydir) get(key string) (entry, bool) {
	k.mu.RLock()
	e, ok := k.entries[key]
	k.mu.RUnlock()
	return e, ok
}

func (k *keydir) delete(key string) {
	k.mu.Lock()
	delete(k.entries, key)
	k.mu.Unlock()
}

// snapshot makes a copy of the map so we can read without holding the lock.
func (k *keydir) snapshot() map[string]entry {
	k.mu.RLock()
	defer k.mu.RUnlock()
	cp := make(map[string]entry, len(k.entries))
	maps.Copy(cp, k.entries)
	return cp
}
