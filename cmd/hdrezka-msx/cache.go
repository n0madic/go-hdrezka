package main

import (
	"sync"
	"time"
)

// cache is a small TTL cache for parsed site metadata. TV navigation walks the
// same pages back and forth, and every miss costs an HTML fetch plus parsing.
type cache struct {
	ttl     time.Duration
	entries sync.Map // string -> cacheEntry
}

type cacheEntry struct {
	value   any
	expires time.Time
}

func newCache(ttl time.Duration) *cache {
	return &cache{ttl: ttl}
}

func (c *cache) get(key string) (any, bool) {
	raw, ok := c.entries.Load(key)
	if !ok {
		return nil, false
	}
	entry := raw.(cacheEntry)
	if time.Now().After(entry.expires) {
		c.entries.Delete(key)
		return nil, false
	}
	return entry.value, true
}

func (c *cache) set(key string, value any) {
	c.entries.Store(key, cacheEntry{value: value, expires: time.Now().Add(c.ttl)})
}

// cached returns the cached value for key, calling load on a miss. Errors are
// not cached, so a failed fetch is retried on the next request.
func cached[T any](c *cache, key string, load func() (T, error)) (T, error) {
	if c != nil && c.ttl > 0 {
		if value, ok := c.get(key); ok {
			if typed, ok := value.(T); ok {
				return typed, nil
			}
		}
	}
	value, err := load()
	if err != nil {
		var zero T
		return zero, err
	}
	if c != nil && c.ttl > 0 {
		c.set(key, value)
	}
	return value, nil
}
