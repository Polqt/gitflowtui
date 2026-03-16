package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type cache struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
	maxSize int
	ttl     time.Duration
}

type cacheEntry struct {
	value      string
	expiresAt  time.Time
	accessedAt time.Time // Track last access time for LRU eviction
}

// get retrieves a value from the cache if it exists and is not expired.
func newCache(maxSize int, ttl time.Duration) *cache {
	if maxSize <= 0 {
		maxSize = 50 // Default max size
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &cache{
		entries: make(map[string]*cacheEntry, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// key generates a unique key for the cache based on the input parts.
// parts are concatenated with a separator to ensure uniqueness and avoid collisions.
func (c *cache) key(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0x00}) // Separator
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *cache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.entries, key)
		return "", false
	}
	entry.accessedAt = time.Now() // Update access time for LRU eviction
	return entry.value, true
}

// set stores a value in the cache with the specified key and sets an expiration time.
func (c *cache) set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict least recently used entry if cache is full
	now := time.Now()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}

	// If still at capacity, evict the least-recently used entry
	if len(c.entries) >= c.maxSize {
		var (
			lruKey  string
			lruTime = time.Now().Add(24 * time.Hour) // Far future time for comparison
		)
		for k, e := range c.entries {
			if e.accessedAt.Before(lruTime) {
				lruTime = e.accessedAt
				lruKey = k
			}
		}
		if lruKey != "" {
			delete(c.entries, lruKey)
		}
	}

	c.entries[key] = &cacheEntry{
		value:      value,
		expiresAt:  now.Add(c.ttl),
		accessedAt: now,
	}
}

// set adds a value to the cache with the specified key and sets an expiration time.
func (c *cache) invalidate(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// flush clears all entries from the cache.
func (c *cache) flush() {
	c.mu.Lock()
	c.entries = make(map[string]*cacheEntry, c.maxSize)
	c.mu.Unlock()
}
