package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type historyCacheEntry struct {
	data      *model.HistoricalPriceResponse
	expiresAt time.Time
}

type HistoryCache struct {
	mu      sync.RWMutex
	entries map[string]historyCacheEntry
	ttl     time.Duration
}

func NewHistoryCache(ttl time.Duration) *HistoryCache {
	return &HistoryCache{
		entries: make(map[string]historyCacheEntry),
		ttl:     ttl,
	}
}

func (c *HistoryCache) cacheKey(ticker, start, end string) string {
	return fmt.Sprintf("%s:%s:%s", ticker, start, end)
}

func (c *HistoryCache) Get(ticker, start, end string) (*model.HistoricalPriceResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[c.cacheKey(ticker, start, end)]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *HistoryCache) Set(ticker, start, end string, data *model.HistoricalPriceResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[c.cacheKey(ticker, start, end)] = historyCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}
