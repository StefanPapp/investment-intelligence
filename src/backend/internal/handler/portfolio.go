package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type PortfolioServiceInterface interface {
	GetPortfolio() (*model.Portfolio, error)
	GetPrice(ticker string) (*model.PriceCache, error)
	GetPriceHistory(ticker, start, end string) (*model.HistoricalPriceResponse, error)
}

type PortfolioHandler struct {
	Svc PortfolioServiceInterface
}

func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	portfolio, err := h.Svc.GetPortfolio()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portfolio)
}

func (h *PortfolioHandler) GetPrice(w http.ResponseWriter, r *http.Request) {
	ticker := chi.URLParam(r, "ticker")
	price, err := h.Svc.GetPrice(ticker)
	if err != nil {
		http.Error(w, `{"error":"price not available for `+ticker+`"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(price)
}

func (h *PortfolioHandler) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	ticker := chi.URLParam(r, "ticker")

	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	// Default: 1 year range ending today
	now := time.Now()
	if end == "" {
		end = now.Format("2006-01-02")
	}
	if start == "" {
		start = now.AddDate(-1, 0, 0).Format("2006-01-02")
	}

	// Validate date formats
	startDate, err := time.Parse("2006-01-02", start)
	if err != nil {
		http.Error(w, `{"error":"invalid start date format, use YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}
	endDate, err := time.Parse("2006-01-02", end)
	if err != nil {
		http.Error(w, `{"error":"invalid end date format, use YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}

	if !startDate.Before(endDate) {
		http.Error(w, `{"error":"start date must be before end date"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.Svc.GetPriceHistory(ticker, start, end)
	if err != nil {
		http.Error(w, `{"error":"price history not available for `+ticker+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
