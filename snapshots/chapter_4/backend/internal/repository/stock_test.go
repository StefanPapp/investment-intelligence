package repository_test

import (
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

func TestStockGetOrCreate_CreatesNew(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := &repository.StockRepo{DB: db}
	stock, err := repo.GetOrCreate("AAPL", "Apple Inc.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stock.Ticker != "AAPL" {
		t.Errorf("expected ticker AAPL, got %s", stock.Ticker)
	}
	if stock.Name != "Apple Inc." {
		t.Errorf("expected name Apple Inc., got %s", stock.Name)
	}
}

func TestStockGetOrCreate_ReturnExisting(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := &repository.StockRepo{DB: db}
	first, err := repo.GetOrCreate("MSFT", "Microsoft Corp.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := repo.GetOrCreate("MSFT", "Microsoft Corp.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same ID, got %s and %s", first.ID, second.ID)
	}
}
