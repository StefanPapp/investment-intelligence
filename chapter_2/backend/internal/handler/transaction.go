package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type TransactionServiceInterface interface {
	Create(req model.CreateTransactionRequest) (*model.Transaction, error)
	GetByID(id uuid.UUID) (*model.Transaction, error)
	List(ticker string) ([]model.Transaction, error)
	Update(id uuid.UUID, req model.UpdateTransactionRequest) (*model.Transaction, error)
	Delete(id uuid.UUID) error
}

type TransactionHandler struct {
	Svc TransactionServiceInterface
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Ticker == "" || req.Shares <= 0 || req.PricePerShare <= 0 || req.TransactionDate == "" {
		http.Error(w, `{"error":"missing required fields"}`, http.StatusBadRequest)
		return
	}

	txn, err := h.Svc.Create(req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	ticker := r.URL.Query().Get("ticker")
	txns, err := h.Svc.List(ticker)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if txns == nil {
		txns = []model.Transaction{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txns)
}

func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	txn, err := h.Svc.GetByID(id)
	if err != nil {
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req model.UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	txn, err := h.Svc.Update(id, req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.Svc.Delete(id); err != nil {
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
