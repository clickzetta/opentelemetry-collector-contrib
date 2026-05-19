// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"sync"
	"time"
)

// RouteEntry holds the resolved routing information for a key,
// including the exporter type and config needed to create/lookup the exporter.
type RouteEntry struct {
	PipelineID     string
	ExporterType   string
	ExporterConfig map[string]any
}

// cacheEntry holds a cached RouteEntry with its expiration time.
type cacheEntry struct {
	entry     *RouteEntry
	expiresAt time.Time
	storedAt  time.Time
}

// RouteCache provides TTL-based caching of RouteEntry entries
// with stale-while-revalidate semantics.
type RouteCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	nowFunc func() time.Time // injected for testability
}

// NewRouteCache creates a new RouteCache with the given TTL.
func NewRouteCache(ttl time.Duration) *RouteCache {
	return &RouteCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
		nowFunc: time.Now,
	}
}

// Get returns the cached RouteEntry for the key.
// Returns (entry, true) if found and fresh.
// Returns (entry, false) if found but expired (stale).
// Returns (nil, false) if not found.
func (c *RouteCache) Get(key string) (*RouteEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ce, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if c.nowFunc().Before(ce.expiresAt) {
		return ce.entry, true
	}

	// Entry exists but is expired — return stale
	return ce.entry, false
}

// Set stores a RouteEntry with the configured TTL.
func (c *RouteCache) Set(key string, entry *RouteEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.nowFunc()
	c.entries[key] = &cacheEntry{
		entry:     entry,
		expiresAt: now.Add(c.ttl),
		storedAt:  now,
	}
}
