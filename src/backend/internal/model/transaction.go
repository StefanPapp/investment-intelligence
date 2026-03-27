package model

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	Buy  TransactionType = "buy"
	Sell TransactionType = "sell"
)

type Transaction struct {
	ID              uuid.UUID       `json:"id"`
	StockID         uuid.UUID       `json:"stock_id"`
	Ticker          string          `json:"ticker"`
	StockName       string          `json:"stock_name"`
	TransactionType TransactionType `json:"transaction_type"`
	Shares          float64         `json:"shares"`
	PricePerShare   float64         `json:"price_per_share"`
	TransactionDate string          `json:"transaction_date"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CreateTransactionRequest struct {
	Ticker          string          `json:"ticker"`
	Name            string          `json:"name,omitempty"`
	TransactionType TransactionType `json:"transaction_type"`
	Shares          float64         `json:"shares"`
	PricePerShare   float64         `json:"price_per_share"`
	TransactionDate string          `json:"transaction_date"`
}

type UpdateTransactionRequest struct {
	TransactionType TransactionType `json:"transaction_type"`
	Shares          float64         `json:"shares"`
	PricePerShare   float64         `json:"price_per_share"`
	TransactionDate string          `json:"transaction_date"`
}
