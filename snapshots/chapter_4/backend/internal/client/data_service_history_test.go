package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
)

func TestDataServiceClient_GetPriceHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/price/AAPL/history" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("start") != "2025-01-01" {
			t.Errorf("unexpected start: %s", r.URL.Query().Get("start"))
		}
		if r.URL.Query().Get("end") != "2025-12-31" {
			t.Errorf("unexpected end: %s", r.URL.Query().Get("end"))
		}
		w.Header().Set("Content-Type", "application/json")
		open := 150.0
		high := 152.0
		low := 149.0
		close := 151.5
		vol := 48000000.0
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ticker":   "AAPL",
			"currency": "USD",
			"interval": "daily",
			"prices": []map[string]interface{}{
				{"date": "2025-01-02", "open": open, "high": high, "low": low, "close": close, "adj_close": close, "volume": vol},
			},
		})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	resp, err := c.GetPriceHistory("AAPL", "2025-01-01", "2025-12-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", resp.Ticker)
	}
	if len(resp.Prices) != 1 {
		t.Fatalf("expected 1 price, got %d", len(resp.Prices))
	}
	if *resp.Prices[0].Open != 150.0 {
		t.Errorf("expected open 150.0, got %f", *resp.Prices[0].Open)
	}
}

func TestDataServiceClient_GetPriceHistory_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail": map[string]interface{}{"error": "No data available", "retryable": false},
		})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	_, err := c.GetPriceHistory("INVALID", "2025-01-01", "2025-12-31")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDataServiceClient_GetPriceHistory_503(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail": map[string]interface{}{"error": "Provider busy", "retryable": true},
		})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	_, err := c.GetPriceHistory("AAPL", "2025-01-01", "2025-12-31")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
