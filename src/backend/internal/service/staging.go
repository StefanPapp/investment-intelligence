package service

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

type StagingService struct {
	StagingRepo *repository.StagingRepo
	StockRepo   *repository.StockRepo
	ImportRepo  *repository.ImportRepo
	DataClient  *client.DataServiceClient
	UploadDir   string
}

// Upload creates an import record and saves the file to UploadDir/importID/filename.
func (s *StagingService) Upload(filename, fileType string, file io.Reader) (*model.UploadResult, error) {
	importID, err := s.StagingRepo.CreateImport(filename, fileType)
	if err != nil {
		return nil, fmt.Errorf("create import: %w", err)
	}

	destDir := filepath.Join(s.UploadDir, importID.String())
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	destPath := filepath.Join(destDir, filename)
	out, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	return &model.UploadResult{ImportID: importID}, nil
}

// Extract calls the data service to extract rows from the uploaded file,
// inserts staging rows, and updates the import status to "extracted".
func (s *StagingService) Extract(importID uuid.UUID) (*model.ImportDetail, error) {
	imp, err := s.StagingRepo.GetImport(importID)
	if err != nil {
		return nil, fmt.Errorf("get import: %w", err)
	}

	filePath := filepath.Join(s.UploadDir, importID.String(), imp.Filename)

	extraction, err := s.DataClient.ExtractFile(filePath, imp.FileType)
	if err != nil {
		return nil, fmt.Errorf("extract file: %w", err)
	}

	if err := s.StagingRepo.InsertStagingRows(importID, extraction.Transactions); err != nil {
		return nil, fmt.Errorf("insert staging rows: %w", err)
	}

	if err := s.StagingRepo.UpdateImportStatus(importID, "extracted"); err != nil {
		return nil, fmt.Errorf("update import status: %w", err)
	}

	return s.GetImport(importID)
}

// GetImport returns the import record along with all its staging rows.
func (s *StagingService) GetImport(importID uuid.UUID) (*model.ImportDetail, error) {
	imp, err := s.StagingRepo.GetImport(importID)
	if err != nil {
		return nil, fmt.Errorf("get import: %w", err)
	}

	rows, err := s.StagingRepo.GetStagingRows(importID)
	if err != nil {
		return nil, fmt.Errorf("get staging rows: %w", err)
	}

	return &model.ImportDetail{
		Import: *imp,
		Rows:   rows,
	}, nil
}

// UpdateRow delegates field-level updates to StagingRepo.
func (s *StagingService) UpdateRow(
	rowID uuid.UUID,
	tradeDate, symbol, side *string,
	quantity, pricePerShare *float64,
) error {
	return s.StagingRepo.UpdateStagingRow(rowID, tradeDate, symbol, side, quantity, pricePerShare)
}

// Confirm processes all "ready" staging rows, upserts them as transactions,
// updates the import status to "confirmed", and removes the upload directory.
func (s *StagingService) Confirm(importID uuid.UUID) (*model.ConfirmResult, error) {
	rows, err := s.StagingRepo.GetStagingRows(importID)
	if err != nil {
		return nil, fmt.Errorf("get staging rows: %w", err)
	}

	result := &model.ConfirmResult{}

	for _, row := range rows {
		if row.Status != "ready" {
			continue
		}

		// All required fields are non-nil when status is "ready".
		ticker := strings.ToUpper(*row.Symbol)
		stock, err := s.StockRepo.GetOrCreate(ticker, ticker)
		if err != nil {
			log.Printf("WARNING: skip staging row %s — stock error: %v", row.ID, err)
			continue
		}

		// Build a deterministic source ID: date_ticker_side_qty_price
		sourceID := fmt.Sprintf("%s_%s_%s_%g_%g",
			*row.TradeDate,
			ticker,
			*row.Side,
			*row.Quantity,
			*row.PricePerShare,
		)

		created, err := s.ImportRepo.UpsertTransaction(
			stock.ID,
			*row.Side,
			*row.Quantity,
			*row.PricePerShare,
			*row.TradeDate,
			"file_import",
			sourceID,
		)
		if err != nil {
			log.Printf("WARNING: skip staging row %s — upsert error: %v", row.ID, err)
			continue
		}

		if created {
			result.Inserted++
		} else {
			result.Duplicates++
		}
	}

	if err := s.StagingRepo.UpdateImportStatus(importID, "confirmed"); err != nil {
		return nil, fmt.Errorf("update import status: %w", err)
	}

	uploadDir := filepath.Join(s.UploadDir, importID.String())
	if err := os.RemoveAll(uploadDir); err != nil {
		log.Printf("WARNING: failed to remove upload dir %s: %v", uploadDir, err)
	}

	return result, nil
}
