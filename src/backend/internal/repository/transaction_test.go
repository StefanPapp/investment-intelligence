package repository_test

import (
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

func TestTransactionCRUD(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	stockRepo := &repository.StockRepo{DB: db}
	txnRepo := &repository.TransactionRepo{DB: db}

	// Create stock first
	stock, err := stockRepo.GetOrCreate("GOOG", "Alphabet Inc.")
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}

	// Create transaction
	req := model.CreateTransactionRequest{
		Ticker:          "GOOG",
		TransactionType: model.Buy,
		Shares:          5,
		PricePerShare:   150.0,
		TransactionDate: "2026-03-01",
	}
	txn, err := txnRepo.Create(stock.ID, req)
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if txn.Shares != 5 {
		t.Errorf("expected 5 shares, got %f", txn.Shares)
	}

	// Get by ID
	fetched, err := txnRepo.GetByID(txn.ID)
	if err != nil {
		t.Fatalf("get transaction: %v", err)
	}
	if fetched.Ticker != "GOOG" {
		t.Errorf("expected ticker GOOG, got %s", fetched.Ticker)
	}

	// List
	list, err := txnRepo.List("")
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(list))
	}

	// Update
	updateReq := model.UpdateTransactionRequest{
		TransactionType: model.Buy,
		Shares:          10,
		PricePerShare:   155.0,
		TransactionDate: "2026-03-02",
	}
	updated, err := txnRepo.Update(txn.ID, updateReq)
	if err != nil {
		t.Fatalf("update transaction: %v", err)
	}
	if updated.Shares != 10 {
		t.Errorf("expected 10 shares, got %f", updated.Shares)
	}

	// Delete
	if err := txnRepo.Delete(txn.ID); err != nil {
		t.Fatalf("delete transaction: %v", err)
	}
	_, err = txnRepo.GetByID(txn.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}
