package cache

import (
	"sync"
	"time"
)

// Item represents a cached item with expiration
type Item struct {
	Value      interface{}
	Expiration int64
}

// Cache is a thread-safe in-memory cache
type Cache struct {
	items map[string]Item
	mu    sync.RWMutex
	ttl   time.Duration
}

// New creates a new cache with the specified default TTL
func New(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]Item),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c
}

// Set stores a value in the cache with the default TTL
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = Item{
		Value:      value,
		Expiration: time.Now().Add(ttl).UnixNano(),
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	if time.Now().UnixNano() > item.Expiration {
		return nil, false
	}

	return item.Value, true
}

// GetOrSet retrieves a value from cache or sets it using the provided function
func (c *Cache) GetOrSet(key string, fn func() (interface{}, error)) (interface{}, error) {
	if value, found := c.Get(key); found {
		return value, nil
	}

	value, err := fn()
	if err != nil {
		return nil, err
	}

	c.Set(key, value)
	return value, nil
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]Item)
}

// cleanup removes expired items periodically
func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now().UnixNano()
		for key, item := range c.items {
			if now > item.Expiration {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// Metrics cache keys
const (
	KeyCPU     = "metrics:cpu"
	KeyMemory  = "metrics:memory"
	KeyDisk    = "metrics:disk"
	KeyNetwork = "metrics:network"
	KeyHost    = "metrics:host"
	KeyAll     = "metrics:all"
)

// MetricsCache is a specialized cache for system metrics
type MetricsCache struct {
	*Cache
}

// NewMetricsCache creates a new metrics cache with a short TTL
func NewMetricsCache() *MetricsCache {
	return &MetricsCache{
		Cache: New(2 * time.Second), // Metrics cached for 2 seconds
	}
}
