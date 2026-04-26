package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type StagingRepo struct {
	DB *sql.DB
}

// CreateImport inserts a new record into the imports table and returns its generated ID.
func (r *StagingRepo) CreateImport(filename, fileType string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.DB.QueryRow(
		`INSERT INTO imports (filename, file_type, status)
		 VALUES ($1, $2, 'pending')
		 RETURNING id`,
		filename, fileType,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create import: %w", err)
	}
	return id, nil
}

// GetImport retrieves an import record by its ID.
func (r *StagingRepo) GetImport(importID uuid.UUID) (*model.Import, error) {
	imp := &model.Import{}
	err := r.DB.QueryRow(
		`SELECT id, filename, file_type, status, created_at, updated_at
		 FROM imports
		 WHERE id = $1`,
		importID,
	).Scan(&imp.ID, &imp.Filename, &imp.FileType, &imp.Status, &imp.CreatedAt, &imp.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get import: %w", err)
	}
	return imp, nil
}

// InsertStagingRows inserts each ExtractedRow into import_staging_rows.
// Status is set to "needs_attention" when warnings are present or any required
// field (TradeDate, Symbol, Side, Quantity, PricePerShare) is nil; otherwise "ready".
func (r *StagingRepo) InsertStagingRows(importID uuid.UUID, rows []model.ExtractedRow) error {
	for i, row := range rows {
		warnings := row.Warnings
		if warnings == nil {
			warnings = []string{}
		}

		// Mark as needs_attention only when truly required fields are missing.
		// Trade date is optional (e.g. holdings imports don't have dates).
		// Informational warnings alone don't block import.
		status := "ready"
		if row.Symbol == nil ||
			row.Side == nil ||
			row.Quantity == nil ||
			row.PricePerShare == nil {
			status = "needs_attention"
		}

		_, err := r.DB.Exec(
			`INSERT INTO import_staging_rows
			 (import_id, trade_date, symbol, side, quantity, price_per_share,
			  currency, fees, account, source_row, warnings, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			importID,
			row.TradeDate,
			row.Symbol,
			row.Side,
			row.Quantity,
			row.PricePerShare,
			row.Currency,
			row.Fees,
			row.Account,
			row.SourceRow,
			pq.StringArray(warnings),
			status,
		)
		if err != nil {
			return fmt.Errorf("insert staging row %d: %w", i, err)
		}
	}
	return nil
}

// GetStagingRows returns all staging rows for an import, ordered by created_at.
func (r *StagingRepo) GetStagingRows(importID uuid.UUID) ([]model.StagingRow, error) {
	result := make([]model.StagingRow, 0)

	rows, err := r.DB.Query(
		`SELECT id, import_id, trade_date, symbol, side, quantity, price_per_share,
		        currency, fees, account, source_row, warnings, status, created_at, updated_at
		 FROM import_staging_rows
		 WHERE import_id = $1
		 ORDER BY created_at`,
		importID,
	)
	if err != nil {
		return nil, fmt.Errorf("query staging rows: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sr model.StagingRow
		if err := rows.Scan(
			&sr.ID, &sr.ImportID, &sr.TradeDate, &sr.Symbol, &sr.Side,
			&sr.Quantity, &sr.PricePerShare, &sr.Currency, &sr.Fees,
			&sr.Account, &sr.SourceRow, &sr.Warnings, &sr.Status,
			&sr.CreatedAt, &sr.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan staging row: %w", err)
		}
		result = append(result, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate staging rows: %w", err)
	}
	return result, nil
}

// UpdateStagingRow updates non-nil fields on a staging row using COALESCE,
// clears warnings to '{}', and sets status to 'ready'.
func (r *StagingRepo) UpdateStagingRow(
	rowID uuid.UUID,
	tradeDate, symbol, side *string,
	quantity, pricePerShare *float64,
) error {
	_, err := r.DB.Exec(
		`UPDATE import_staging_rows
		 SET trade_date     = COALESCE($2, trade_date),
		     symbol         = COALESCE($3, symbol),
		     side           = COALESCE($4, side),
		     quantity       = COALESCE($5, quantity),
		     price_per_share = COALESCE($6, price_per_share),
		     warnings       = '{}',
		     status         = 'ready',
		     updated_at     = NOW()
		 WHERE id = $1`,
		rowID, tradeDate, symbol, side, quantity, pricePerShare,
	)
	if err != nil {
		return fmt.Errorf("update staging row: %w", err)
	}
	return nil
}

// UpdateImportStatus updates the status field of an import record.
func (r *StagingRepo) UpdateImportStatus(importID uuid.UUID, status string) error {
	_, err := r.DB.Exec(
		`UPDATE imports SET status = $2, updated_at = NOW() WHERE id = $1`,
		importID, status,
	)
	if err != nil {
		return fmt.Errorf("update import status: %w", err)
	}
	return nil
}

// DeleteImport deletes an import record; cascades to import_staging_rows.
func (r *StagingRepo) DeleteImport(importID uuid.UUID) error {
	result, err := r.DB.Exec(`DELETE FROM imports WHERE id = $1`, importID)
	if err != nil {
		return fmt.Errorf("delete import: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete import rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
