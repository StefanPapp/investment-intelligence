package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

// mockTransactionService implements a minimal mock for handler testing
type mockTransactionService struct {
	createFn func(req model.CreateTransactionRequest) (*model.Transaction, error)
	listFn   func(ticker string) ([]model.Transaction, error)
}

func (m *mockTransactionService) Create(req model.CreateTransactionRequest) (*model.Transaction, error) {
	return m.createFn(req)
}

func (m *mockTransactionService) GetByID(id uuid.UUID) (*model.Transaction, error) {
	panic("not implemented")
}

func (m *mockTransactionService) List(ticker string) ([]model.Transaction, error) {
	return m.listFn(ticker)
}

func (m *mockTransactionService) Update(id uuid.UUID, req model.UpdateTransactionRequest) (*model.Transaction, error) {
	panic("not implemented")
}

func (m *mockTransactionService) Delete(id uuid.UUID) error {
	panic("not implemented")
}

func TestCreateTransactionHandler_ValidInput(t *testing.T) {
	mock := &mockTransactionService{
		createFn: func(req model.CreateTransactionRequest) (*model.Transaction, error) {
			return &model.Transaction{
				Ticker:          req.Ticker,
				TransactionType: req.TransactionType,
				Shares:          req.Shares,
				PricePerShare:   req.PricePerShare,
				TransactionDate: req.TransactionDate,
			}, nil
		},
	}

	h := &handler.TransactionHandler{Svc: mock}

	body, _ := json.Marshal(model.CreateTransactionRequest{
		Ticker:          "AAPL",
		TransactionType: "buy",
		Shares:          10,
		PricePerShare:   185.50,
		TransactionDate: "2026-03-01",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result model.Transaction
	json.NewDecoder(w.Body).Decode(&result)
	if result.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", result.Ticker)
	}
}

func TestListTransactionsHandler(t *testing.T) {
	mock := &mockTransactionService{
		listFn: func(ticker string) ([]model.Transaction, error) {
			return []model.Transaction{
				{Ticker: "AAPL", Shares: 10},
			}, nil
		},
	}

	h := &handler.TransactionHandler{Svc: mock}

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
