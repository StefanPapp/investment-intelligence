package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
)

func TestDataServiceClient_GetPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/price/AAPL" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ticker":     "AAPL",
			"price":      192.30,
			"currency":   "USD",
			"fetched_at": "2026-03-05T10:00:00Z",
		})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	price, err := c.GetPrice("AAPL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", price.Ticker)
	}
	if price.Price != 192.30 {
		t.Errorf("expected 192.30, got %f", price.Price)
	}
}

func TestDataServiceClient_GetPrice_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"detail": "Ticker not found"})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	_, err := c.GetPrice("INVALID")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
