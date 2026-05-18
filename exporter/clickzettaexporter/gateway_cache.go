// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"sync"
	"time"
)

// cacheEntry holds a cached TenantConfig with its expiration time.
type cacheEntry struct {
	config    *TenantConfig
	expiresAt time.Time
}

// KeyCache provides TTL-based caching of TenantConfig entries with
// stale-while-revalidate semantics.
type KeyCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

// NewKeyCache creates a new KeyCache with the given TTL.
func NewKeyCache(ttl time.Duration) *KeyCache {
	return &KeyCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

// Get returns the cached TenantConfig for the key.
// Returns (config, true) if found and not expired (fresh).
// Returns (config, false) if found but expired (stale).
// Returns (nil, false) if not found.
func (c *KeyCache) Get(key string) (*TenantConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().Before(entry.expiresAt) {
		return entry.config, true
	}
	// Entry exists but is expired — return stale
	return entry.config, false
}

// Set stores a TenantConfig with the configured TTL.
func (c *KeyCache) Set(key string, config *TenantConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		config:    config,
		expiresAt: time.Now().Add(c.ttl),
	}
}
