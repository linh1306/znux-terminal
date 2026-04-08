package suggest

import (
	"sync"
	"time"
)

// CacheEntry holds a cached suggestion result
type CacheEntry struct {
	Suggestions []interface{}
	Expiry      time.Time
}

// Cache provides TTL-based caching for suggestion generators
type Cache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
}

// NewCache creates a new cache with the given TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves cached suggestions for a key
func (c *Cache) Get(key string) ([]interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.Expiry) {
		return nil, false
	}
	return entry.Suggestions, true
}

// Set stores suggestions for a key
func (c *Cache) Set(key string, suggestions []interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		Suggestions: suggestions,
		Expiry:      time.Now().Add(c.ttl),
	}
}

// Invalidate removes a key from cache
func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes all entries from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

// CachedGenerator wraps a generator function with caching
type CachedGenerator struct {
	generator func() []interface{}
	cache     *Cache
	key       string
}

// NewCachedGenerator creates a new cached generator
func NewCachedGenerator(key string, generator func() []interface{}, ttl time.Duration) *CachedGenerator {
	return &CachedGenerator{
		generator: generator,
		cache:     NewCache(ttl),
		key:       key,
	}
}

// Get returns cached or fresh suggestions
func (cg *CachedGenerator) Get() []interface{} {
	if cached, ok := cg.cache.Get(cg.key); ok {
		return cached
	}
	suggestions := cg.generator()
	cg.cache.Set(cg.key, suggestions)
	return suggestions
}

// Invalidate invalidates the cache for this generator
func (cg *CachedGenerator) Invalidate() {
	cg.cache.Invalidate(cg.key)
}
