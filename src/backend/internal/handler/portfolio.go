package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type PortfolioServiceInterface interface {
	GetPortfolio() (*model.Portfolio, error)
	GetPrice(ticker string) (*model.PriceCache, error)
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
