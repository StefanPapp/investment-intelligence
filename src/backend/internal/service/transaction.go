package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

type TransactionService struct {
	StockRepo *repository.StockRepo
	TxnRepo   *repository.TransactionRepo
}

func (s *TransactionService) Create(req model.CreateTransactionRequest) (*model.Transaction, error) {
	req.Ticker = strings.ToUpper(req.Ticker)
	if req.Name == "" {
		req.Name = req.Ticker
	}

	stock, err := s.StockRepo.GetOrCreate(req.Ticker, req.Name)
	if err != nil {
		return nil, fmt.Errorf("get or create stock: %w", err)
	}

	txn, err := s.TxnRepo.Create(stock.ID, req)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}
	txn.Ticker = stock.Ticker
	txn.StockName = stock.Name
	return txn, nil
}

func (s *TransactionService) GetByID(id uuid.UUID) (*model.Transaction, error) {
	txn, err := s.TxnRepo.GetByID(id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	return txn, err
}

func (s *TransactionService) List(ticker string) ([]model.Transaction, error) {
	return s.TxnRepo.List(strings.ToUpper(ticker))
}

func (s *TransactionService) Update(id uuid.UUID, req model.UpdateTransactionRequest) (*model.Transaction, error) {
	txn, err := s.TxnRepo.Update(id, req)
	if err != nil {
		return nil, fmt.Errorf("update transaction: %w", err)
	}
	return txn, nil
}

func (s *TransactionService) Delete(id uuid.UUID) error {
	err := s.TxnRepo.Delete(id)
	if err == sql.ErrNoRows {
		return fmt.Errorf("transaction not found")
	}
	return err
}
