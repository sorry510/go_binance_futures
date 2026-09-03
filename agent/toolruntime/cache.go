package toolruntime

import (
	"sync"
	"time"
)

type cacheEntry struct {
	raw       []byte
	expiresAt time.Time
}

type memoryCache struct {
	mu    sync.Mutex
	items map[string]cacheEntry
}

func newMemoryCache() *memoryCache { return &memoryCache{items: map[string]cacheEntry{}} }

func (cache *memoryCache) get(key string, now time.Time) ([]byte, bool) {
	if cache == nil || key == "" {
		return nil, false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	entry, ok := cache.items[key]
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.After(now) {
		delete(cache.items, key)
		return nil, false
	}
	return append([]byte(nil), entry.raw...), true
}

func (cache *memoryCache) set(key string, raw []byte, ttl time.Duration, now time.Time) {
	if cache == nil || key == "" || len(raw) == 0 || ttl <= 0 {
		return
	}
	cache.mu.Lock()
	cache.items[key] = cacheEntry{raw: append([]byte(nil), raw...), expiresAt: now.Add(ttl)}
	cache.mu.Unlock()
}
