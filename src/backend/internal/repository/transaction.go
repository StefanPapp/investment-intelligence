package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type TransactionRepo struct {
	DB *sql.DB
}

func (r *TransactionRepo) Create(stockID uuid.UUID, req model.CreateTransactionRequest) (*model.Transaction, error) {
	t := &model.Transaction{}
	err := r.DB.QueryRow(
		`INSERT INTO transactions (stock_id, transaction_type, shares, price_per_share, transaction_date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, stock_id, transaction_type, shares, price_per_share, transaction_date, created_at, updated_at`,
		stockID, req.TransactionType, req.Shares, req.PricePerShare, req.TransactionDate,
	).Scan(&t.ID, &t.StockID, &t.TransactionType, &t.Shares, &t.PricePerShare, &t.TransactionDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	return t, nil
}

func (r *TransactionRepo) GetByID(id uuid.UUID) (*model.Transaction, error) {
	t := &model.Transaction{}
	err := r.DB.QueryRow(
		`SELECT t.id, t.stock_id, s.ticker, s.name, t.transaction_type, t.shares, t.price_per_share,
		        t.transaction_date, t.created_at, t.updated_at
		 FROM transactions t
		 JOIN stocks s ON s.id = t.stock_id
		 WHERE t.id = $1`, id,
	).Scan(&t.ID, &t.StockID, &t.Ticker, &t.StockName, &t.TransactionType, &t.Shares, &t.PricePerShare,
		&t.TransactionDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TransactionRepo) List(ticker string) ([]model.Transaction, error) {
	query := `SELECT t.id, t.stock_id, s.ticker, s.name, t.transaction_type, t.shares, t.price_per_share,
	                 t.transaction_date, t.created_at, t.updated_at
	          FROM transactions t
	          JOIN stocks s ON s.id = t.stock_id`
	var args []interface{}
	if ticker != "" {
		query += " WHERE s.ticker = $1"
		args = append(args, ticker)
	}
	query += " ORDER BY t.transaction_date DESC"

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var txns []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.StockID, &t.Ticker, &t.StockName, &t.TransactionType,
			&t.Shares, &t.PricePerShare, &t.TransactionDate, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

func (r *TransactionRepo) Update(id uuid.UUID, req model.UpdateTransactionRequest) (*model.Transaction, error) {
	t := &model.Transaction{}
	err := r.DB.QueryRow(
		`UPDATE transactions
		 SET transaction_type = $2, shares = $3, price_per_share = $4, transaction_date = $5, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, stock_id, transaction_type, shares, price_per_share, transaction_date, created_at, updated_at`,
		id, req.TransactionType, req.Shares, req.PricePerShare, req.TransactionDate,
	).Scan(&t.ID, &t.StockID, &t.TransactionType, &t.Shares, &t.PricePerShare, &t.TransactionDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update transaction: %w", err)
	}
	return t, nil
}

func (r *TransactionRepo) Delete(id uuid.UUID) error {
	result, err := r.DB.Exec(`DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
