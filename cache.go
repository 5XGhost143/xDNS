package main

import (
	"sync"
	"time"
)

type cacheEntry struct {
	response  []byte
	timestamp time.Time
}

type Cache struct {
	entries map[string]*cacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
}

func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
	go c.cleanup()
	return c
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	responseCopy := make([]byte, len(entry.response))
	copy(responseCopy, entry.response)
	return responseCopy, true
}

func (c *Cache) Set(key string, response []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	responseCopy := make([]byte, len(response))
	copy(responseCopy, response)

	c.entries[key] = &cacheEntry{
		response:  responseCopy,
		timestamp: time.Now(),
	}
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.Sub(entry.timestamp) > c.ttl {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}
