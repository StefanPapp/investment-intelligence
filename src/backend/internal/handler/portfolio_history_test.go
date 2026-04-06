package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type mockPortfolioService struct {
	getPortfolioFn    func() (*model.Portfolio, error)
	getPriceFn        func(ticker string) (*model.PriceCache, error)
	getPriceHistoryFn func(ticker, start, end string) (*model.HistoricalPriceResponse, error)
}

func (m *mockPortfolioService) GetPortfolio() (*model.Portfolio, error) {
	if m.getPortfolioFn != nil {
		return m.getPortfolioFn()
	}
	return nil, nil
}

func (m *mockPortfolioService) GetPrice(ticker string) (*model.PriceCache, error) {
	if m.getPriceFn != nil {
		return m.getPriceFn(ticker)
	}
	return nil, nil
}

func (m *mockPortfolioService) GetPriceHistory(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
	if m.getPriceHistoryFn != nil {
		return m.getPriceHistoryFn(ticker, start, end)
	}
	return nil, nil
}

func newTestRouter(h *handler.PortfolioHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/prices/{ticker}/history", h.GetPriceHistory)
	return r
}

func TestGetPriceHistory_Success(t *testing.T) {
	open := 150.0
	mock := &mockPortfolioService{
		getPriceHistoryFn: func(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
			return &model.HistoricalPriceResponse{
				Ticker:   ticker,
				Currency: "USD",
				Interval: "daily",
				Prices: []model.HistoricalPrice{
					{Date: "2025-01-02", Open: &open},
				},
			}, nil
		},
	}

	h := &handler.PortfolioHandler{Svc: mock}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/AAPL/history?start=2025-01-01&end=2025-12-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.HistoricalPriceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", resp.Ticker)
	}
}

func TestGetPriceHistory_DefaultDates(t *testing.T) {
	mock := &mockPortfolioService{
		getPriceHistoryFn: func(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
			if start == "" || end == "" {
				t.Error("expected default dates to be set")
			}
			return &model.HistoricalPriceResponse{Ticker: ticker, Prices: []model.HistoricalPrice{}}, nil
		},
	}

	h := &handler.PortfolioHandler{Svc: mock}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/AAPL/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPriceHistory_InvalidDateFormat(t *testing.T) {
	h := &handler.PortfolioHandler{Svc: &mockPortfolioService{}}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/AAPL/history?start=not-a-date&end=2025-12-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetPriceHistory_StartAfterEnd(t *testing.T) {
	h := &handler.PortfolioHandler{Svc: &mockPortfolioService{}}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/AAPL/history?start=2025-12-31&end=2025-01-01", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetPriceHistory_ServiceError(t *testing.T) {
	mock := &mockPortfolioService{
		getPriceHistoryFn: func(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
			return nil, fmt.Errorf("data service returned 404 for ticker INVALID history")
		},
	}

	h := &handler.PortfolioHandler{Svc: mock}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/INVALID/history?start=2025-01-01&end=2025-12-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
