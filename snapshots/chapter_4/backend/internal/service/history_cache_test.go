package service

import (
	"testing"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

func TestHistoryCache_GetMiss(t *testing.T) {
	cache := NewHistoryCache(15 * time.Minute)
	_, ok := cache.Get("AAPL", "2025-01-01", "2025-12-31")
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestHistoryCache_SetAndGet(t *testing.T) {
	cache := NewHistoryCache(15 * time.Minute)
	resp := &model.HistoricalPriceResponse{
		Ticker:   "AAPL",
		Currency: "USD",
		Interval: "daily",
		Prices:   []model.HistoricalPrice{{Date: "2025-01-02"}},
	}

	cache.Set("AAPL", "2025-01-01", "2025-12-31", resp)

	got, ok := cache.Get("AAPL", "2025-01-01", "2025-12-31")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", got.Ticker)
	}
	if len(got.Prices) != 1 {
		t.Errorf("expected 1 price, got %d", len(got.Prices))
	}
}

func TestHistoryCache_Expiry(t *testing.T) {
	cache := NewHistoryCache(1 * time.Millisecond)
	resp := &model.HistoricalPriceResponse{Ticker: "AAPL"}
	cache.Set("AAPL", "2025-01-01", "2025-12-31", resp)

	time.Sleep(5 * time.Millisecond)

	_, ok := cache.Get("AAPL", "2025-01-01", "2025-12-31")
	if ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestHistoryCache_DifferentKeysIndependent(t *testing.T) {
	cache := NewHistoryCache(15 * time.Minute)
	resp1 := &model.HistoricalPriceResponse{Ticker: "AAPL"}
	resp2 := &model.HistoricalPriceResponse{Ticker: "GOOG"}

	cache.Set("AAPL", "2025-01-01", "2025-12-31", resp1)
	cache.Set("GOOG", "2025-01-01", "2025-12-31", resp2)

	got, ok := cache.Get("AAPL", "2025-01-01", "2025-12-31")
	if !ok || got.Ticker != "AAPL" {
		t.Fatal("expected AAPL cache hit")
	}
	got, ok = cache.Get("GOOG", "2025-01-01", "2025-12-31")
	if !ok || got.Ticker != "GOOG" {
		t.Fatal("expected GOOG cache hit")
	}
}
