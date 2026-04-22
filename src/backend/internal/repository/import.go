package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type ImportRepo struct {
	DB *sql.DB
}

// FindBySourceID looks up a transaction by its source and source_id.
// Returns sql.ErrNoRows if not found.
func (r *ImportRepo) FindBySourceID(source, sourceID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.DB.QueryRow(
		`SELECT id FROM transactions WHERE source = $1 AND source_id = $2`,
		source, sourceID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// UpsertTransaction inserts or updates a transaction identified by source + source_id.
// Returns true if a new row was created, false if an existing row was updated.
func (r *ImportRepo) UpsertTransaction(
	stockID uuid.UUID,
	transactionType string,
	shares float64,
	pricePerShare float64,
	transactionDate string,
	source string,
	sourceID string,
) (bool, error) {
	existingID, err := r.FindBySourceID(source, sourceID)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("find by source id: %w", err)
	}

	if err == sql.ErrNoRows {
		// Insert new
		_, err = r.DB.Exec(
			`INSERT INTO transactions (stock_id, transaction_type, shares, price_per_share, transaction_date, source, source_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			stockID, transactionType, shares, pricePerShare, transactionDate, source, sourceID,
		)
		if err != nil {
			return false, fmt.Errorf("insert transaction: %w", err)
		}
		return true, nil
	}

	// Update existing
	_, err = r.DB.Exec(
		`UPDATE transactions
		 SET stock_id = $2, transaction_type = $3, shares = $4, price_per_share = $5, transaction_date = $6, updated_at = NOW()
		 WHERE id = $1`,
		existingID, stockID, transactionType, shares, pricePerShare, transactionDate,
	)
	if err != nil {
		return false, fmt.Errorf("update transaction: %w", err)
	}
	return false, nil
}
