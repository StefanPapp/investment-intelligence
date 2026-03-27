package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type StockRepo struct {
	DB *sql.DB
}

func (r *StockRepo) GetOrCreate(ticker, name string) (*model.Stock, error) {
	stock, err := r.GetByTicker(ticker)
	if err == nil {
		return stock, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("get stock: %w", err)
	}

	s := &model.Stock{}
	err = r.DB.QueryRow(
		`INSERT INTO stocks (ticker, name) VALUES ($1, $2)
		 RETURNING id, ticker, name, created_at`,
		ticker, name,
	).Scan(&s.ID, &s.Ticker, &s.Name, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert stock: %w", err)
	}
	return s, nil
}

func (r *StockRepo) GetByTicker(ticker string) (*model.Stock, error) {
	s := &model.Stock{}
	err := r.DB.QueryRow(
		`SELECT id, ticker, name, created_at FROM stocks WHERE ticker = $1`, ticker,
	).Scan(&s.ID, &s.Ticker, &s.Name, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *StockRepo) GetByID(id uuid.UUID) (*model.Stock, error) {
	s := &model.Stock{}
	err := r.DB.QueryRow(
		`SELECT id, ticker, name, created_at FROM stocks WHERE id = $1`, id,
	).Scan(&s.ID, &s.Ticker, &s.Name, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}
