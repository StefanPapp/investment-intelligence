# File-based Holdings Import via LLM Extraction — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users upload a CSV or screenshot of a broker statement, extract transactions via LLM, review/edit them in an editable table, and commit approved rows to the database idempotently.

**Architecture:** File upload goes through Next.js server action to Go backend, which stores it temporarily and forwards it to Python data-service for LLM extraction. Extracted rows land in a staging table for review. On user confirmation, Go commits staging rows to `transactions`. The flow is: Upload -> Extract -> Review/Edit -> Confirm.

**Tech Stack:** Go (chi, multipart handling, staging repo), Python (FastAPI, LangChain structured output, Claude/OpenAI vision), Next.js (server actions, drag-and-drop upload, editable review table), Postgres (import_staging table).

---

## File Structure

### New files

| File                                                    | Responsibility                                                        |
| ------------------------------------------------------- | --------------------------------------------------------------------- |
| **Go backend**                                          |                                                                       |
| `backend/migrations/000003_add_import_staging.up.sql`   | Create `imports` and `import_staging_rows` tables                     |
| `backend/migrations/000003_add_import_staging.down.sql` | Drop staging tables                                                   |
| `backend/internal/model/import.go`                      | Import, StagingRow, UploadResult, ExtractResult, ConfirmResult models |
| `backend/internal/repository/staging.go`                | StagingRepo: CRUD for imports + staging rows                          |
| `backend/internal/handler/upload.go`                    | UploadHandler: upload, get import, patch row, confirm                 |
| `backend/internal/service/staging.go`                   | StagingService: orchestrates upload -> extract -> confirm             |
| **Python data-service**                                 |                                                                       |
| `data-service/src/models/extract.py`                    | ExtractedTransaction, ExtractionResponse Pydantic models              |
| `data-service/src/services/extraction_service.py`       | ExtractionService: LangChain structured output extraction             |
| `data-service/src/routers/extract.py`                   | POST /extract endpoint                                                |
| **Next.js frontend**                                    |                                                                       |
| `frontend/components/file-dropzone.tsx`                 | Drag-and-drop file upload component                                   |
| `frontend/components/review-table.tsx`                  | Editable staging row table with Ready/Needs attention/Skipped groups  |
| `frontend/app/import/[importId]/page.tsx`               | Review page for a specific import                                     |
| **Tests**                                               |                                                                       |
| `data-service/tests/test_extract.py`                    | Extraction endpoint tests                                             |
| `backend/internal/handler/upload_test.go`               | Upload handler tests                                                  |
| `backend/internal/repository/staging_test.go`           | Staging repo tests                                                    |

### Modified files

| File                                      | Changes                                                             |
| ----------------------------------------- | ------------------------------------------------------------------- |
| `backend/cmd/server/main.go`              | Add StagingRepo, StagingService, UploadHandler; mount new routes    |
| `backend/internal/client/data_service.go` | Add `ExtractFile()` method                                          |
| `data-service/src/main.py`                | Mount extract router                                                |
| `data-service/pyproject.toml`             | Add `langchain`, `langchain-anthropic` (or `langchain-openai`) deps |
| `frontend/lib/api.ts`                     | Add upload, getImport, patchRow, confirmImport functions            |
| `frontend/app/actions/import.ts`          | Add uploadFileAction, patchRowAction, confirmImportAction           |
| `frontend/app/import/page.tsx`            | Add file upload section below Alpaca section                        |
| `docker-compose.yml`                      | Pass `ANTHROPIC_API_KEY` to data-service                            |
| `.env.example`                            | Add `ANTHROPIC_API_KEY`                                             |

---

## Slice 1 — Upload Endpoint and Modal

### Task 1: Migration for staging tables

**Files:**

- Create: `backend/migrations/000003_add_import_staging.up.sql`
- Create: `backend/migrations/000003_add_import_staging.down.sql`

- [ ] **Step 1: Write the up migration**

```sql
-- 000003_add_import_staging.up.sql
CREATE TABLE imports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    filename TEXT NOT NULL,
    file_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE import_staging_rows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    import_id UUID NOT NULL REFERENCES imports(id) ON DELETE CASCADE,
    trade_date TEXT,
    symbol TEXT,
    side TEXT,
    quantity NUMERIC(18, 8),
    price_per_share NUMERIC(18, 8),
    currency TEXT DEFAULT 'USD',
    fees NUMERIC(18, 8) DEFAULT 0,
    account TEXT,
    source_row TEXT,
    warnings TEXT[] DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'ready',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_staging_rows_import_id ON import_staging_rows(import_id);
```

- [ ] **Step 2: Write the down migration**

```sql
-- 000003_add_import_staging.down.sql
DROP TABLE IF EXISTS import_staging_rows;
DROP TABLE IF EXISTS imports;
```

- [ ] **Step 3: Verify migration applies**

Run: `docker-compose up --build -d && docker-compose logs backend --tail 5`
Expected: "Migrations complete" in logs

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/000003_add_import_staging.up.sql backend/migrations/000003_add_import_staging.down.sql
git commit -m "feat(backend): add imports and import_staging_rows tables"
```

---

### Task 2: Go models for file import

**Files:**

- Create: `backend/internal/model/import.go`

- [ ] **Step 1: Define the models**

```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Import struct {
	ID        uuid.UUID `json:"id"`
	Filename  string    `json:"filename"`
	FileType  string    `json:"file_type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StagingRow struct {
	ID            uuid.UUID      `json:"id"`
	ImportID      uuid.UUID      `json:"import_id"`
	TradeDate     *string        `json:"trade_date"`
	Symbol        *string        `json:"symbol"`
	Side          *string        `json:"side"`
	Quantity      *float64       `json:"quantity"`
	PricePerShare *float64       `json:"price_per_share"`
	Currency      string         `json:"currency"`
	Fees          float64        `json:"fees"`
	Account       *string        `json:"account"`
	SourceRow     *string        `json:"source_row"`
	Warnings      pq.StringArray `json:"warnings"`
	Status        string         `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type ImportDetail struct {
	Import Import       `json:"import"`
	Rows   []StagingRow `json:"rows"`
}

type UploadResult struct {
	ImportID uuid.UUID `json:"import_id"`
}

type ConfirmResult struct {
	Inserted   int `json:"inserted"`
	Duplicates int `json:"duplicates"`
}

// ExtractedRow is what the Python data-service returns per extracted transaction.
type ExtractedRow struct {
	TradeDate     *string  `json:"trade_date"`
	Symbol        *string  `json:"symbol"`
	Side          *string  `json:"side"`
	Quantity      *float64 `json:"quantity"`
	PricePerShare *float64 `json:"price_per_share"`
	Currency      string   `json:"currency"`
	Fees          float64  `json:"fees"`
	Account       *string  `json:"account"`
	SourceRow     *string  `json:"source_row"`
	Warnings      []string `json:"warnings"`
}

type ExtractionResponse struct {
	Transactions []ExtractedRow `json:"transactions"`
	Skipped      []SkippedRow   `json:"skipped"`
}

type SkippedRow struct {
	SourceRow string `json:"source_row"`
	Reason    string `json:"reason"`
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/model/import.go
git commit -m "feat(backend): add file import models"
```

---

### Task 3: Staging repository

**Files:**

- Create: `backend/internal/repository/staging.go`
- Create: `backend/internal/repository/staging_test.go`

- [ ] **Step 1: Write tests for StagingRepo**

```go
// staging_test.go — build-only test to verify compilation + method signatures
package repository

import (
	"testing"
)

func TestStagingRepoCompiles(t *testing.T) {
	// Verify StagingRepo has the expected methods (compilation check).
	// Integration tests against a real DB would use testcontainers.
	var _ interface {
		CreateImport(filename, fileType string) (string, error)
		GetImport(importID string) (*importRecord, error)
		InsertStagingRows(importID string, rows []stagingRowInput) error
		GetStagingRows(importID string) ([]stagingRowRecord, error)
		UpdateStagingRow(rowID string, updates map[string]interface{}) error
		UpdateImportStatus(importID, status string) error
		DeleteImport(importID string) error
	}
	_ = &StagingRepo{}
	// This test passes if the code compiles with all required methods.
	// If a method is missing or has wrong signature, this won't compile.
	_ = t // use t
}
```

Note: This is a structural/compilation test. The real validation happens via the integration tests in Task 10. Remove the interface assertion once integration tests exist.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/repository/ -run TestStagingRepoCompiles -v`
Expected: FAIL — StagingRepo not defined

- [ ] **Step 3: Implement StagingRepo**

```go
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

func (r *StagingRepo) CreateImport(filename, fileType string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.DB.QueryRow(
		`INSERT INTO imports (filename, file_type, status) VALUES ($1, $2, 'pending')
		 RETURNING id`,
		filename, fileType,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create import: %w", err)
	}
	return id, nil
}

func (r *StagingRepo) GetImport(importID uuid.UUID) (*model.Import, error) {
	imp := &model.Import{}
	err := r.DB.QueryRow(
		`SELECT id, filename, file_type, status, created_at, updated_at
		 FROM imports WHERE id = $1`, importID,
	).Scan(&imp.ID, &imp.Filename, &imp.FileType, &imp.Status, &imp.CreatedAt, &imp.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get import: %w", err)
	}
	return imp, nil
}

func (r *StagingRepo) InsertStagingRows(importID uuid.UUID, rows []model.ExtractedRow) error {
	for _, row := range rows {
		warnings := pq.StringArray(row.Warnings)
		if warnings == nil {
			warnings = pq.StringArray{}
		}
		status := "ready"
		if len(row.Warnings) > 0 || row.TradeDate == nil || row.Symbol == nil || row.Side == nil || row.Quantity == nil || row.PricePerShare == nil {
			status = "needs_attention"
		}

		_, err := r.DB.Exec(
			`INSERT INTO import_staging_rows
			 (import_id, trade_date, symbol, side, quantity, price_per_share, currency, fees, account, source_row, warnings, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			importID, row.TradeDate, row.Symbol, row.Side, row.Quantity,
			row.PricePerShare, row.Currency, row.Fees, row.Account,
			row.SourceRow, warnings, status,
		)
		if err != nil {
			return fmt.Errorf("insert staging row: %w", err)
		}
	}
	return nil
}

func (r *StagingRepo) GetStagingRows(importID uuid.UUID) ([]model.StagingRow, error) {
	rows, err := r.DB.Query(
		`SELECT id, import_id, trade_date, symbol, side, quantity, price_per_share,
		        currency, fees, account, source_row, warnings, status, created_at, updated_at
		 FROM import_staging_rows WHERE import_id = $1
		 ORDER BY created_at`, importID,
	)
	if err != nil {
		return nil, fmt.Errorf("get staging rows: %w", err)
	}
	defer rows.Close()

	result := make([]model.StagingRow, 0)
	for rows.Next() {
		var sr model.StagingRow
		err := rows.Scan(
			&sr.ID, &sr.ImportID, &sr.TradeDate, &sr.Symbol, &sr.Side,
			&sr.Quantity, &sr.PricePerShare, &sr.Currency, &sr.Fees,
			&sr.Account, &sr.SourceRow, &sr.Warnings, &sr.Status,
			&sr.CreatedAt, &sr.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan staging row: %w", err)
		}
		result = append(result, sr)
	}
	return result, nil
}

func (r *StagingRepo) UpdateStagingRow(rowID uuid.UUID, tradeDate, symbol, side *string, quantity, pricePerShare *float64) error {
	_, err := r.DB.Exec(
		`UPDATE import_staging_rows
		 SET trade_date = COALESCE($2, trade_date),
		     symbol = COALESCE($3, symbol),
		     side = COALESCE($4, side),
		     quantity = COALESCE($5, quantity),
		     price_per_share = COALESCE($6, price_per_share),
		     warnings = '{}',
		     status = 'ready',
		     updated_at = NOW()
		 WHERE id = $1`,
		rowID, tradeDate, symbol, side, quantity, pricePerShare,
	)
	if err != nil {
		return fmt.Errorf("update staging row: %w", err)
	}
	return nil
}

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

func (r *StagingRepo) DeleteImport(importID uuid.UUID) error {
	_, err := r.DB.Exec(`DELETE FROM imports WHERE id = $1`, importID)
	if err != nil {
		return fmt.Errorf("delete import: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Fix compile test to match actual signatures**

Replace the compile test with a simpler version now that the real types exist:

```go
package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

func TestStagingRepoCompiles(t *testing.T) {
	repo := &StagingRepo{}
	// Verify method signatures compile. We don't call them (no DB).
	_ = repo
	var _ func(string, string) (uuid.UUID, error) = repo.CreateImport
	var _ func(uuid.UUID) (*model.Import, error) = repo.GetImport
	var _ func(uuid.UUID, []model.ExtractedRow) error = repo.InsertStagingRows
	var _ func(uuid.UUID) ([]model.StagingRow, error) = repo.GetStagingRows
	var _ func(uuid.UUID, string) error = repo.UpdateImportStatus
	var _ func(uuid.UUID) error = repo.DeleteImport
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./internal/repository/ -run TestStagingRepoCompiles -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/staging.go backend/internal/repository/staging_test.go
git commit -m "feat(backend): add staging repository for file imports"
```

---

### Task 4: Upload handler and staging service

**Files:**

- Create: `backend/internal/handler/upload.go`
- Create: `backend/internal/service/staging.go`

- [ ] **Step 1: Write StagingService**

```go
package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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

func (s *StagingService) Upload(filename, fileType string, file io.Reader) (*model.UploadResult, error) {
	importID, err := s.StagingRepo.CreateImport(filename, fileType)
	if err != nil {
		return nil, fmt.Errorf("create import record: %w", err)
	}

	// Save file to disk keyed by import_id
	dir := filepath.Join(s.UploadDir, importID.String())
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	dst, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	return &model.UploadResult{ImportID: importID}, nil
}

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

	rows, err := s.StagingRepo.GetStagingRows(importID)
	if err != nil {
		return nil, fmt.Errorf("get staging rows: %w", err)
	}

	imp.Status = "extracted"
	return &model.ImportDetail{Import: *imp, Rows: rows}, nil
}

func (s *StagingService) GetImport(importID uuid.UUID) (*model.ImportDetail, error) {
	imp, err := s.StagingRepo.GetImport(importID)
	if err != nil {
		return nil, fmt.Errorf("get import: %w", err)
	}

	rows, err := s.StagingRepo.GetStagingRows(importID)
	if err != nil {
		return nil, fmt.Errorf("get staging rows: %w", err)
	}

	return &model.ImportDetail{Import: *imp, Rows: rows}, nil
}

func (s *StagingService) UpdateRow(rowID uuid.UUID, tradeDate, symbol, side *string, quantity, pricePerShare *float64) error {
	return s.StagingRepo.UpdateStagingRow(rowID, tradeDate, symbol, side, quantity, pricePerShare)
}

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
		if row.Symbol == nil || row.TradeDate == nil || row.Side == nil || row.Quantity == nil || row.PricePerShare == nil {
			continue
		}

		ticker := *row.Symbol
		stock, err := s.StockRepo.GetOrCreate(ticker, ticker)
		if err != nil {
			continue
		}

		// Use (trade_date, symbol, side, quantity, price_per_share) as idempotency key
		sourceID := fmt.Sprintf("%s_%s_%s_%.8f_%.8f", *row.TradeDate, ticker, *row.Side, *row.Quantity, *row.PricePerShare)

		created, err := s.ImportRepo.UpsertTransaction(
			stock.ID, *row.Side, *row.Quantity, *row.PricePerShare,
			*row.TradeDate, "file_import", sourceID,
		)
		if err != nil {
			continue
		}

		if created {
			result.Inserted++
		} else {
			result.Duplicates++
		}
	}

	// Clean up staging data and source file
	_ = s.StagingRepo.UpdateImportStatus(importID, "confirmed")
	_ = os.RemoveAll(filepath.Join(s.UploadDir, importID.String()))

	return result, nil
}
```

- [ ] **Step 2: Write UploadHandler**

```go
package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var allowedExtensions = map[string]bool{
	".csv":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".pdf":  true,
}

type StagingServiceInterface interface {
	Upload(filename, fileType string, file interface{ Read([]byte) (int, error) }) (*uploadResult, error)
	Extract(importID uuid.UUID) (*importDetail, error)
	GetImport(importID uuid.UUID) (*importDetail, error)
	UpdateRow(rowID uuid.UUID, tradeDate, symbol, side *string, quantity, pricePerShare *float64) error
	Confirm(importID uuid.UUID) (*confirmResult, error)
}

type UploadHandler struct {
	Svc interface {
		Upload(filename, fileType string, file interface{ Read([]byte) (int, error) }) (*model.UploadResult, error)
		Extract(importID uuid.UUID) (*model.ImportDetail, error)
		GetImport(importID uuid.UUID) (*model.ImportDetail, error)
		UpdateRow(rowID uuid.UUID, tradeDate, symbol, side *string, quantity, pricePerShare *float64) error
		Confirm(importID uuid.UUID) (*model.ConfirmResult, error)
	}
}
```

Note: use the actual model import. Full file below:

```go
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/service"
)

var allowedExtensions = map[string]bool{
	".csv":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".pdf":  true,
}

const maxUploadSize = 10 << 20 // 10 MB

type UploadHandler struct {
	Svc *service.StagingService
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, `{"error":"file too large (max 10MB)"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"missing file field"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		http.Error(w, `{"error":"unsupported file type"}`, http.StatusBadRequest)
		return
	}

	result, err := h.Svc.Upload(header.Filename, ext, file)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Trigger extraction immediately
	detail, err := h.Svc.Extract(result.ImportID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *UploadHandler) GetImport(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "importId"))
	if err != nil {
		http.Error(w, `{"error":"invalid import id"}`, http.StatusBadRequest)
		return
	}

	detail, err := h.Svc.GetImport(id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *UploadHandler) PatchRow(w http.ResponseWriter, r *http.Request) {
	rowID, err := uuid.Parse(chi.URLParam(r, "rowId"))
	if err != nil {
		http.Error(w, `{"error":"invalid row id"}`, http.StatusBadRequest)
		return
	}

	var body struct {
		TradeDate     *string  `json:"trade_date"`
		Symbol        *string  `json:"symbol"`
		Side          *string  `json:"side"`
		Quantity      *float64 `json:"quantity"`
		PricePerShare *float64 `json:"price_per_share"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := h.Svc.UpdateRow(rowID, body.TradeDate, body.Symbol, body.Side, body.Quantity, body.PricePerShare); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UploadHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "importId"))
	if err != nil {
		http.Error(w, `{"error":"invalid import id"}`, http.StatusBadRequest)
		return
	}

	result, err := h.Svc.Confirm(id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 3: Fix io import in StagingService.Upload signature**

The `Upload` method takes `io.Reader` not `*multipart.File`. Ensure `file` parameter in `service/staging.go` is `io.Reader`:

```go
func (s *StagingService) Upload(filename, fileType string, file io.Reader) (*model.UploadResult, error) {
```

- [ ] **Step 4: Wire into router**

Add to `cmd/server/main.go` after the existing repo/service/handler setup:

```go
// In the repos section:
stagingRepo := &repository.StagingRepo{DB: db}

// In the services section:
stagingSvc := &service.StagingService{
    StagingRepo: stagingRepo,
    StockRepo:   stockRepo,
    ImportRepo:  importRepo,
    DataClient:  dataClient,
    UploadDir:   "/tmp/imports",
}

// In the handlers section:
uploadHandler := &handler.UploadHandler{Svc: stagingSvc}

// In the router, inside r.Route("/api", ...) add:
r.Post("/imports/upload", uploadHandler.Upload)
r.Get("/imports/{importId}", uploadHandler.GetImport)
r.Patch("/imports/{importId}/rows/{rowId}", uploadHandler.PatchRow)
r.Post("/imports/{importId}/confirm", uploadHandler.Confirm)
```

- [ ] **Step 5: Verify build**

Run: `cd backend && go build ./...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/upload.go backend/internal/service/staging.go backend/cmd/server/main.go
git commit -m "feat(backend): add upload handler and staging service for file imports"
```

---

## Slice 2 — LLM Extraction via LangChain

### Task 5: Add LangChain dependencies to data-service

**Files:**

- Modify: `data-service/pyproject.toml`

- [ ] **Step 1: Add dependencies**

Add to `dependencies` in `pyproject.toml`:

```toml
"langchain>=0.3.0",
"langchain-anthropic>=0.3.0",
```

- [ ] **Step 2: Install**

Run: `cd data-service && uv sync`
Expected: packages resolve and install

- [ ] **Step 3: Commit**

```bash
git add data-service/pyproject.toml data-service/uv.lock
git commit -m "chore(data-service): add langchain and langchain-anthropic deps"
```

---

### Task 6: Extraction Pydantic models

**Files:**

- Create: `data-service/src/models/extract.py`

- [ ] **Step 1: Define models**

```python
from pydantic import BaseModel, Field

from src.models.price import JsonDecimal


class ExtractedTransaction(BaseModel):
    """A single transaction extracted from a broker statement."""

    trade_date: str | None = Field(
        default=None,
        description="Trade date in YYYY-MM-DD format. null if ambiguous.",
    )
    symbol: str | None = Field(
        default=None, description="Ticker symbol, e.g. AAPL, MSFT."
    )
    side: str | None = Field(
        default=None, description="'buy' or 'sell'."
    )
    quantity: JsonDecimal | None = Field(
        default=None, description="Number of shares, positive."
    )
    price_per_share: JsonDecimal | None = Field(
        default=None, description="Price per share, positive, in transaction currency."
    )
    currency: str = Field(
        default="USD", description="ISO 4217 currency code."
    )
    fees: JsonDecimal = Field(
        default=0, description="Transaction fees, positive, default 0."
    )
    account: str | None = Field(
        default=None, description="Account identifier if present."
    )
    source_row: str | None = Field(
        default=None, description="Verbatim source text for audit."
    )
    warnings: list[str] = Field(
        default_factory=list,
        description="Warnings about this row, e.g. ambiguous date.",
    )


class SkippedRow(BaseModel):
    """A row excluded from transactions (dividend, split, fee, etc.)."""

    source_row: str = Field(description="Verbatim source text.")
    reason: str = Field(description="Why this row was skipped: 'dividend', 'split', 'fee', etc.")


class ExtractionResult(BaseModel):
    """Result of extracting transactions from a broker statement."""

    transactions: list[ExtractedTransaction] = Field(default_factory=list)
    skipped: list[SkippedRow] = Field(default_factory=list)
```

- [ ] **Step 2: Commit**

```bash
git add data-service/src/models/extract.py
git commit -m "feat(data-service): add extraction Pydantic models"
```

---

### Task 7: Extraction service with LangChain structured output

**Files:**

- Create: `data-service/src/services/extraction_service.py`

- [ ] **Step 1: Write the extraction service**

```python
import asyncio
import base64
import logging
import os
import mimetypes
from pathlib import Path

from langchain_anthropic import ChatAnthropic
from langchain_core.messages import HumanMessage

from src.models.extract import ExtractionResult

logger = logging.getLogger(__name__)

SYSTEM_PROMPT = """You are a financial data extractor. Given a broker statement (CSV text or image),
extract every buy and sell transaction into structured JSON.

Rules:
- Dates: Accept MM/DD/YYYY, YYYY-MM-DD, or DD.MM.YYYY. Normalize to YYYY-MM-DD.
  If ambiguous (e.g. 03/04/2024 with no other signal), set trade_date to null and add a warning.
- Numbers: Accept . or , as decimal separator. Strip currency symbols. Return numeric values.
- Currency: Preserve per-row currency. Default USD if not stated.
- Partial fills: Treat as separate transactions.
- Missing required fields: Set to null and add a warning.
- Dividends, splits, fees, corporate actions: Exclude from transactions. Put in skipped list
  with the reason (e.g. "dividend", "split", "fee").
- source_row: Include the verbatim source text for each row for audit purposes.
- Do NOT guess. If uncertain, flag with a warning."""


class ExtractionService:
    """Extracts transactions from broker statements using LangChain structured output."""

    def __init__(self) -> None:
        api_key = os.getenv("ANTHROPIC_API_KEY", "")
        if not api_key:
            raise ValueError("ANTHROPIC_API_KEY is required")

        self._llm = ChatAnthropic(
            model="claude-sonnet-4-20250514",
            api_key=api_key,
            max_tokens=4096,
        ).with_structured_output(ExtractionResult)

    async def extract_csv(self, content: str) -> ExtractionResult:
        """Extract transactions from CSV text content."""
        message = HumanMessage(content=f"Extract transactions from this broker CSV:\n\n{content}")
        result = await asyncio.to_thread(self._llm.invoke, [message])
        logger.info(
            "Extracted %d transactions, %d skipped from CSV",
            len(result.transactions),
            len(result.skipped),
        )
        return result

    async def extract_image(self, file_path: str) -> ExtractionResult:
        """Extract transactions from an image (PNG, JPG, PDF)."""
        path = Path(file_path)
        mime_type = mimetypes.guess_type(str(path))[0] or "image/png"
        image_data = base64.standard_b64encode(path.read_bytes()).decode("utf-8")

        message = HumanMessage(
            content=[
                {
                    "type": "image_url",
                    "image_url": {"url": f"data:{mime_type};base64,{image_data}"},
                },
                {
                    "type": "text",
                    "text": "Extract all buy and sell transactions from this broker statement image.",
                },
            ]
        )
        result = await asyncio.to_thread(self._llm.invoke, [message])
        logger.info(
            "Extracted %d transactions, %d skipped from image",
            len(result.transactions),
            len(result.skipped),
        )
        return result
```

Note: `asyncio.to_thread` wraps the blocking LangChain `.invoke()` call per the project rules (blocking I/O in async funcs). The system prompt is injected via the LLM's system parameter — update `ChatAnthropic(...)` constructor:

```python
self._llm = ChatAnthropic(
    model="claude-sonnet-4-20250514",
    api_key=api_key,
    max_tokens=4096,
).bind(system=SYSTEM_PROMPT).with_structured_output(ExtractionResult)
```

If `bind(system=...)` is not supported, pass the system prompt as the first message in the invoke call instead. Check langchain-anthropic docs via Context7 before implementation.

- [ ] **Step 2: Commit**

```bash
git add data-service/src/services/extraction_service.py
git commit -m "feat(data-service): add LLM extraction service with LangChain structured output"
```

---

### Task 8: Extract endpoint in data-service

**Files:**

- Create: `data-service/src/routers/extract.py`
- Modify: `data-service/src/main.py`

- [ ] **Step 1: Write the router**

```python
import logging
from pathlib import Path

from fastapi import APIRouter, HTTPException, UploadFile

from src.models.extract import ExtractionResult
from src.services.extraction_service import ExtractionService

logger = logging.getLogger(__name__)

router = APIRouter()

try:
    extraction_service = ExtractionService()
except ValueError:
    extraction_service = None
    logger.warning("ANTHROPIC_API_KEY not configured — /extract will return 503")


@router.post("/extract", response_model=ExtractionResult)
async def extract(file: UploadFile) -> ExtractionResult:
    """Extract transactions from an uploaded broker statement file."""
    if extraction_service is None:
        raise HTTPException(status_code=503, detail="ANTHROPIC_API_KEY not configured")

    if file.filename is None:
        raise HTTPException(status_code=400, detail="filename required")

    ext = Path(file.filename).suffix.lower()
    content_bytes = await file.read()

    if ext == ".csv":
        text = content_bytes.decode("utf-8", errors="replace")
        return await extraction_service.extract_csv(text)

    if ext in (".png", ".jpg", ".jpeg", ".pdf"):
        # Save temp file for image extraction
        import tempfile
        with tempfile.NamedTemporaryFile(suffix=ext, delete=False) as tmp:
            tmp.write(content_bytes)
            tmp_path = tmp.name
        try:
            return await extraction_service.extract_image(tmp_path)
        finally:
            Path(tmp_path).unlink(missing_ok=True)

    raise HTTPException(status_code=400, detail=f"unsupported file type: {ext}")
```

- [ ] **Step 2: Mount router in main.py**

Add to `data-service/src/main.py`:

```python
from src.routers.extract import router as extract_router

app.include_router(extract_router)
```

- [ ] **Step 3: Commit**

```bash
git add data-service/src/routers/extract.py data-service/src/main.py
git commit -m "feat(data-service): add POST /extract endpoint for LLM extraction"
```

---

### Task 9: Go client method to call /extract

**Files:**

- Modify: `backend/internal/client/data_service.go`

- [ ] **Step 1: Add ExtractFile method**

Add to `data_service.go`:

```go
func (c *DataServiceClient) ExtractFile(filePath, fileType string) (*model.ExtractionResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file for extraction: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy file to form: %w", err)
	}
	writer.Close()

	// LLM extraction can be slow
	extractClient := &http.Client{Timeout: 120 * time.Second}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/extract", c.baseURL), body)
	if err != nil {
		return nil, fmt.Errorf("create extract request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := extractClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("extract file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &DataServiceError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("data service returned %d for extract", resp.StatusCode),
		}
	}

	var result model.ExtractionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode extraction response: %w", err)
	}
	return &result, nil
}
```

Add imports at top of file: `"bytes"`, `"io"`, `"mime/multipart"`, `"os"`, `"path/filepath"`.

- [ ] **Step 2: Verify build**

Run: `cd backend && go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/internal/client/data_service.go
git commit -m "feat(backend): add ExtractFile client method for LLM extraction"
```

---

### Task 10: Extraction endpoint tests

**Files:**

- Create: `data-service/tests/test_extract.py`

- [ ] **Step 1: Write tests**

```python
from unittest.mock import AsyncMock, patch

import pytest
from httpx import ASGITransport, AsyncClient

from src.main import app
from src.models.extract import ExtractionResult, ExtractedTransaction, SkippedRow


@pytest.fixture
def fidelity_csv_content():
    return (
        "Run Date,Action,Symbol,Quantity,Price,Amount\n"
        "03/15/2024,YOU BOUGHT,AAPL,10,172.50,-1725.00\n"
        "04/01/2024,DIVIDEND,AAPL,,0.24,2.40\n"
    )


@pytest.fixture
def mock_extraction_result():
    return ExtractionResult(
        transactions=[
            ExtractedTransaction(
                trade_date="2024-03-15",
                symbol="AAPL",
                side="buy",
                quantity=10,
                price_per_share=172.50,
                currency="USD",
                fees=0,
                source_row="03/15/2024,YOU BOUGHT,AAPL,10,172.50,-1725.00",
            ),
        ],
        skipped=[
            SkippedRow(
                source_row="04/01/2024,DIVIDEND,AAPL,,0.24,2.40",
                reason="dividend",
            ),
        ],
    )


async def test_extract_csv_success(fidelity_csv_content, mock_extraction_result):
    with patch("src.routers.extract.extraction_service") as mock_svc:
        mock_svc.extract_csv = AsyncMock(return_value=mock_extraction_result)
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.post(
                "/extract",
                files={"file": ("fidelity.csv", fidelity_csv_content.encode(), "text/csv")},
            )
    assert resp.status_code == 200
    data = resp.json()
    assert len(data["transactions"]) == 1
    assert data["transactions"][0]["symbol"] == "AAPL"
    assert data["transactions"][0]["side"] == "buy"
    assert data["transactions"][0]["trade_date"] == "2024-03-15"
    assert len(data["skipped"]) == 1
    assert data["skipped"][0]["reason"] == "dividend"


async def test_extract_no_api_key():
    with patch("src.routers.extract.extraction_service", None):
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.post(
                "/extract",
                files={"file": ("test.csv", b"header\ndata", "text/csv")},
            )
    assert resp.status_code == 503
    assert "ANTHROPIC_API_KEY" in resp.json()["detail"]


async def test_extract_unsupported_file_type():
    with patch("src.routers.extract.extraction_service") as mock_svc:
        mock_svc.extract_csv = AsyncMock()
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.post(
                "/extract",
                files={"file": ("test.docx", b"data", "application/vnd.openxmlformats")},
            )
    assert resp.status_code == 400
    assert "unsupported" in resp.json()["detail"]
```

- [ ] **Step 2: Run tests**

Run: `cd data-service && uv run pytest tests/test_extract.py -v`
Expected: 3 tests PASS

- [ ] **Step 3: Commit**

```bash
git add data-service/tests/test_extract.py
git commit -m "test(data-service): add extraction endpoint tests"
```

---

### Task 11: Docker compose env for ANTHROPIC_API_KEY

**Files:**

- Modify: `docker-compose.yml`
- Modify: `.env.example`

- [ ] **Step 1: Add env var to docker-compose.yml**

In the `data-service` service environment section, add:

```yaml
ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY:-}
```

- [ ] **Step 2: Add to .env.example**

Add after the Alpaca section:

```
# LLM extraction (Chapter 4)
ANTHROPIC_API_KEY=sk-ant-...
```

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml .env.example
git commit -m "chore: pass ANTHROPIC_API_KEY to data-service container"
```

---

## Slice 3 — Review Table (Frontend)

### Task 12: Frontend API client functions for imports

**Files:**

- Modify: `frontend/lib/api.ts`

- [ ] **Step 1: Add interfaces and functions**

Add to `lib/api.ts`:

```typescript
export interface StagingRow {
  id: string;
  import_id: string;
  trade_date: string | null;
  symbol: string | null;
  side: string | null;
  quantity: number | null;
  price_per_share: number | null;
  currency: string;
  fees: number;
  account: string | null;
  source_row: string | null;
  warnings: string[];
  status: string;
}

export interface ImportDetail {
  import: {
    id: string;
    filename: string;
    file_type: string;
    status: string;
  };
  rows: StagingRow[];
}

export interface ConfirmResult {
  inserted: number;
  duplicates: number;
}

export async function uploadFile(file: File): Promise<ImportDetail> {
  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(`${BACKEND_URL}/api/imports/upload`, {
    method: "POST",
    body: formData,
    // Do NOT set Content-Type — browser sets it with boundary for multipart
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
  return res.json();
}

export async function getImport(importId: string): Promise<ImportDetail> {
  return apiFetch<ImportDetail>(`/api/imports/${importId}`);
}

export async function patchStagingRow(
  importId: string,
  rowId: string,
  updates: Partial<
    Pick<
      StagingRow,
      "trade_date" | "symbol" | "side" | "quantity" | "price_per_share"
    >
  >,
): Promise<void> {
  const res = await fetch(
    `${BACKEND_URL}/api/imports/${importId}/rows/${rowId}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(updates),
    },
  );
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
}

export async function confirmImport(importId: string): Promise<ConfirmResult> {
  return apiFetch<ConfirmResult>(`/api/imports/${importId}/confirm`, {
    method: "POST",
  });
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/lib/api.ts
git commit -m "feat(frontend): add import API client functions"
```

---

### Task 13: Server actions for file import

**Files:**

- Modify: `frontend/app/actions/import.ts`

- [ ] **Step 1: Add server actions**

Add to `app/actions/import.ts`:

```typescript
import {
  importAlpaca,
  uploadFile,
  patchStagingRow,
  confirmImport,
} from "@/lib/api";

// ... existing importAlpacaAction stays ...

export async function uploadFileAction(formData: FormData): Promise<{
  success: boolean;
  importId?: string;
  rows?: import("@/lib/api").StagingRow[];
  error?: string;
}> {
  const file = formData.get("file") as File;
  if (!file || file.size === 0) {
    return { success: false, error: "No file selected" };
  }

  try {
    const detail = await uploadFile(file);
    revalidatePath("/import");
    return {
      success: true,
      importId: detail.import.id,
      rows: detail.rows,
    };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Upload failed",
    };
  }
}

export async function patchRowAction(
  importId: string,
  rowId: string,
  updates: Record<string, unknown>,
): Promise<{ success: boolean; error?: string }> {
  try {
    await patchStagingRow(importId, rowId, updates);
    return { success: true };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Update failed",
    };
  }
}

export async function confirmImportAction(importId: string): Promise<{
  success: boolean;
  inserted?: number;
  duplicates?: number;
  error?: string;
}> {
  try {
    const result = await confirmImport(importId);
    revalidatePath("/");
    revalidatePath("/transactions");
    return {
      success: true,
      inserted: result.inserted,
      duplicates: result.duplicates,
    };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Confirm failed",
    };
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/app/actions/import.ts
git commit -m "feat(frontend): add server actions for file upload, patch, confirm"
```

---

### Task 14: File dropzone component

**Files:**

- Create: `frontend/components/file-dropzone.tsx`

- [ ] **Step 1: Write the component**

```tsx
"use client";

import { useCallback, useState } from "react";

interface FileDropzoneProps {
  onFileSelected: (file: File) => void;
  disabled?: boolean;
}

const ACCEPTED_TYPES = [".csv", ".png", ".jpg", ".jpeg", ".pdf"];
const MAX_SIZE = 10 * 1024 * 1024; // 10 MB

export default function FileDropzone({
  onFileSelected,
  disabled = false,
}: FileDropzoneProps) {
  const [dragActive, setDragActive] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const validateAndSelect = useCallback(
    (file: File) => {
      setError(null);
      const ext = file.name.substring(file.name.lastIndexOf(".")).toLowerCase();
      if (!ACCEPTED_TYPES.includes(ext)) {
        setError(
          `Unsupported file type: ${ext}. Accepted: ${ACCEPTED_TYPES.join(", ")}`,
        );
        return;
      }
      if (file.size > MAX_SIZE) {
        setError("File too large (max 10 MB)");
        return;
      }
      onFileSelected(file);
    },
    [onFileSelected],
  );

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragActive(false);
    if (disabled) return;
    const file = e.dataTransfer.files[0];
    if (file) validateAndSelect(file);
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault();
    if (!disabled) setDragActive(true);
  }

  function handleDragLeave() {
    setDragActive(false);
  }

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) validateAndSelect(file);
  }

  return (
    <div>
      <div
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
          disabled
            ? "border-gray-200 bg-gray-50 cursor-not-allowed"
            : dragActive
              ? "border-blue-500 bg-blue-50"
              : "border-gray-300 hover:border-gray-400 cursor-pointer"
        }`}
      >
        <p className="text-gray-600 text-sm mb-2">
          Drag and drop a file here, or click to browse
        </p>
        <p className="text-gray-400 text-xs">
          CSV, PNG, JPG, or PDF (max 10 MB)
        </p>
        <input
          type="file"
          accept={ACCEPTED_TYPES.join(",")}
          onChange={handleChange}
          disabled={disabled}
          className="hidden"
          id="file-upload"
        />
        <label
          htmlFor="file-upload"
          className={`inline-block mt-3 px-4 py-2 text-sm rounded-md ${
            disabled
              ? "bg-gray-300 text-gray-500 cursor-not-allowed"
              : "bg-blue-600 text-white hover:bg-blue-700 cursor-pointer"
          }`}
        >
          Choose File
        </label>
      </div>
      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/components/file-dropzone.tsx
git commit -m "feat(frontend): add file dropzone component"
```

---

### Task 15: Review table component

**Files:**

- Create: `frontend/components/review-table.tsx`

- [ ] **Step 1: Write the component**

```tsx
"use client";

import { useState } from "react";
import type { StagingRow } from "@/lib/api";
import { patchRowAction } from "@/app/actions/import";

interface ReviewTableProps {
  importId: string;
  rows: StagingRow[];
  onRowUpdated: (rowId: string, updates: Partial<StagingRow>) => void;
}

export default function ReviewTable({
  importId,
  rows,
  onRowUpdated,
}: ReviewTableProps) {
  const ready = rows.filter((r) => r.status === "ready");
  const needsAttention = rows.filter((r) => r.status === "needs_attention");
  const skipped = rows.filter((r) => r.status === "skipped");

  return (
    <div className="space-y-6">
      {needsAttention.length > 0 && (
        <RowGroup
          title="Needs Attention"
          rows={needsAttention}
          importId={importId}
          onRowUpdated={onRowUpdated}
          bgColor="bg-amber-50"
          borderColor="border-amber-200"
        />
      )}
      {ready.length > 0 && (
        <RowGroup
          title="Ready"
          rows={ready}
          importId={importId}
          onRowUpdated={onRowUpdated}
          bgColor="bg-green-50"
          borderColor="border-green-200"
        />
      )}
      {skipped.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-gray-500 mb-2">
            Skipped ({skipped.length})
          </h3>
          <div className="bg-gray-50 rounded-lg border border-gray-200 p-4">
            {skipped.map((row) => (
              <p key={row.id} className="text-xs text-gray-500">
                {row.source_row}
              </p>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function RowGroup({
  title,
  rows,
  importId,
  onRowUpdated,
  bgColor,
  borderColor,
}: {
  title: string;
  rows: StagingRow[];
  importId: string;
  onRowUpdated: (rowId: string, updates: Partial<StagingRow>) => void;
  bgColor: string;
  borderColor: string;
}) {
  return (
    <div>
      <h3 className="text-sm font-semibold text-gray-700 mb-2">
        {title} ({rows.length})
      </h3>
      <div
        className={`${bgColor} rounded-lg border ${borderColor} overflow-x-auto`}
      >
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left text-gray-500">
              <th className="py-2 px-3">Date</th>
              <th className="py-2 px-3">Symbol</th>
              <th className="py-2 px-3">Side</th>
              <th className="py-2 px-3 text-right">Qty</th>
              <th className="py-2 px-3 text-right">Price</th>
              <th className="py-2 px-3">Currency</th>
              <th className="py-2 px-3">Warnings</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <EditableRow
                key={row.id}
                row={row}
                importId={importId}
                onUpdated={onRowUpdated}
              />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function EditableRow({
  row,
  importId,
  onUpdated,
}: {
  row: StagingRow;
  importId: string;
  onUpdated: (rowId: string, updates: Partial<StagingRow>) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [tradeDate, setTradeDate] = useState(row.trade_date ?? "");
  const [symbol, setSymbol] = useState(row.symbol ?? "");
  const [side, setSide] = useState(row.side ?? "buy");
  const [quantity, setQuantity] = useState(row.quantity?.toString() ?? "");
  const [price, setPrice] = useState(row.price_per_share?.toString() ?? "");
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    const updates = {
      trade_date: tradeDate || null,
      symbol: symbol || null,
      side: side || null,
      quantity: quantity ? parseFloat(quantity) : null,
      price_per_share: price ? parseFloat(price) : null,
    };
    const result = await patchRowAction(importId, row.id, updates);
    if (result.success) {
      onUpdated(row.id, { ...updates, status: "ready", warnings: [] });
      setEditing(false);
    }
    setSaving(false);
  }

  if (editing) {
    return (
      <tr className="border-b">
        <td className="py-2 px-3">
          <input
            type="date"
            value={tradeDate}
            onChange={(e) => setTradeDate(e.target.value)}
            className="border rounded px-1 py-0.5 text-xs w-28"
          />
        </td>
        <td className="py-2 px-3">
          <input
            value={symbol}
            onChange={(e) => setSymbol(e.target.value.toUpperCase())}
            className="border rounded px-1 py-0.5 text-xs w-16"
          />
        </td>
        <td className="py-2 px-3">
          <select
            value={side}
            onChange={(e) => setSide(e.target.value)}
            className="border rounded px-1 py-0.5 text-xs"
          >
            <option value="buy">buy</option>
            <option value="sell">sell</option>
          </select>
        </td>
        <td className="py-2 px-3 text-right">
          <input
            type="number"
            step="any"
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
            className="border rounded px-1 py-0.5 text-xs w-20 text-right"
          />
        </td>
        <td className="py-2 px-3 text-right">
          <input
            type="number"
            step="any"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            className="border rounded px-1 py-0.5 text-xs w-24 text-right"
          />
        </td>
        <td className="py-2 px-3">{row.currency}</td>
        <td className="py-2 px-3">
          <button
            onClick={handleSave}
            disabled={saving}
            className="text-blue-600 hover:underline text-xs mr-2"
          >
            {saving ? "Saving..." : "Save"}
          </button>
          <button
            onClick={() => setEditing(false)}
            className="text-gray-500 hover:underline text-xs"
          >
            Cancel
          </button>
        </td>
      </tr>
    );
  }

  return (
    <tr className="border-b hover:bg-white/50">
      <td className="py-2 px-3">
        {row.trade_date ?? <span className="text-red-500">missing</span>}
      </td>
      <td className="py-2 px-3 font-medium">
        {row.symbol ?? <span className="text-red-500">missing</span>}
      </td>
      <td className="py-2 px-3">
        {row.side && (
          <span
            className={`px-2 py-0.5 rounded text-xs ${
              row.side === "buy"
                ? "bg-green-100 text-green-800"
                : "bg-red-100 text-red-800"
            }`}
          >
            {row.side.toUpperCase()}
          </span>
        )}
      </td>
      <td className="py-2 px-3 text-right">{row.quantity}</td>
      <td className="py-2 px-3 text-right">
        {row.price_per_share !== null ? (
          new Intl.NumberFormat("en-US", {
            style: "currency",
            currency: row.currency || "USD",
          }).format(row.price_per_share)
        ) : (
          <span className="text-red-500">missing</span>
        )}
      </td>
      <td className="py-2 px-3">{row.currency}</td>
      <td className="py-2 px-3">
        {row.warnings.length > 0 && (
          <span className="text-amber-600 text-xs">
            {row.warnings.join("; ")}
          </span>
        )}
        <button
          onClick={() => setEditing(true)}
          className="text-blue-600 hover:underline text-xs ml-2"
        >
          Edit
        </button>
      </td>
    </tr>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/components/review-table.tsx
git commit -m "feat(frontend): add editable review table component"
```

---

### Task 16: Import page with file upload and review flow

**Files:**

- Modify: `frontend/app/import/page.tsx`
- Create: `frontend/app/import/[importId]/page.tsx`

- [ ] **Step 1: Update import page to include file upload section**

Rewrite `app/import/page.tsx` to add a file upload section below the existing Alpaca section:

```tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { importAlpacaAction, uploadFileAction } from "@/app/actions/import";
import FileDropzone from "@/components/file-dropzone";
import type { StagingRow } from "@/lib/api";

export default function ImportPage() {
  const router = useRouter();

  // Alpaca state
  const [alpacaLoading, setAlpacaLoading] = useState(false);
  const [alpacaResult, setAlpacaResult] = useState<{
    success: boolean;
    created?: number;
    updated?: number;
    total?: number;
    error?: string;
  } | null>(null);

  // File upload state
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  async function handleAlpacaImport() {
    setAlpacaLoading(true);
    setAlpacaResult(null);
    const res = await importAlpacaAction();
    setAlpacaResult(res);
    setAlpacaLoading(false);
  }

  async function handleFileUpload() {
    if (!selectedFile) return;
    setUploading(true);
    setUploadError(null);

    const formData = new FormData();
    formData.append("file", selectedFile);

    const result = await uploadFileAction(formData);
    if (result.success && result.importId) {
      router.push(`/import/${result.importId}`);
    } else {
      setUploadError(result.error ?? "Upload failed");
    }
    setUploading(false);
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Import Transactions</h1>

      {/* Alpaca section */}
      <div className="bg-white rounded-lg shadow-sm border p-6 mb-6">
        <h2 className="text-lg font-semibold mb-2">Alpaca</h2>
        <p className="text-gray-600 text-sm mb-4">
          Import your filled orders from Alpaca. Existing orders are matched by
          order ID and updated if changed.
        </p>
        <button
          onClick={handleAlpacaImport}
          disabled={alpacaLoading}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {alpacaLoading ? "Importing..." : "Import from Alpaca"}
        </button>
        {alpacaResult && (
          <div
            className={`mt-4 p-4 rounded-md text-sm ${
              alpacaResult.success
                ? "bg-green-50 text-green-800 border border-green-200"
                : "bg-red-50 text-red-800 border border-red-200"
            }`}
          >
            {alpacaResult.success ? (
              <p>
                Imported {alpacaResult.total} order
                {alpacaResult.total !== 1 ? "s" : ""}
                {" \u2014 "}
                {alpacaResult.created} created, {alpacaResult.updated} updated.
              </p>
            ) : (
              <p>Import failed: {alpacaResult.error}</p>
            )}
          </div>
        )}
      </div>

      {/* File upload section */}
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <h2 className="text-lg font-semibold mb-2">File Import</h2>
        <p className="text-gray-600 text-sm mb-4">
          Upload a broker statement (CSV or screenshot). Transactions will be
          extracted using AI and presented for review before import.
        </p>
        <FileDropzone
          onFileSelected={(file) => {
            setSelectedFile(file);
            setUploadError(null);
          }}
          disabled={uploading}
        />
        {selectedFile && (
          <div className="mt-4 flex items-center gap-3">
            <span className="text-sm text-gray-700">{selectedFile.name}</span>
            <button
              onClick={handleFileUpload}
              disabled={uploading}
              className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {uploading ? "Extracting..." : "Upload & Extract"}
            </button>
          </div>
        )}
        {uploadError && (
          <div className="mt-4 p-4 rounded-md text-sm bg-red-50 text-red-800 border border-red-200">
            <p>{uploadError}</p>
          </div>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Create review page**

Create `app/import/[importId]/page.tsx`:

```tsx
"use client";

import { use, useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import ReviewTable from "@/components/review-table";
import { confirmImportAction } from "@/app/actions/import";
import type { StagingRow } from "@/lib/api";
import { getImport } from "@/lib/api";

export default function ImportReviewPage({
  params,
}: {
  params: Promise<{ importId: string }>;
}) {
  const { importId } = use(params);
  const router = useRouter();
  const [rows, setRows] = useState<StagingRow[]>([]);
  const [filename, setFilename] = useState("");
  const [loading, setLoading] = useState(true);
  const [confirming, setConfirming] = useState(false);
  const [confirmResult, setConfirmResult] = useState<{
    inserted: number;
    duplicates: number;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getImport(importId)
      .then((detail) => {
        setRows(detail.rows);
        setFilename(detail.import.filename);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [importId]);

  const handleRowUpdated = useCallback(
    (rowId: string, updates: Partial<StagingRow>) => {
      setRows((prev) =>
        prev.map((r) => (r.id === rowId ? { ...r, ...updates } : r)),
      );
    },
    [],
  );

  async function handleConfirm() {
    setConfirming(true);
    setError(null);
    const result = await confirmImportAction(importId);
    if (result.success) {
      setConfirmResult({
        inserted: result.inserted ?? 0,
        duplicates: result.duplicates ?? 0,
      });
    } else {
      setError(result.error ?? "Confirm failed");
    }
    setConfirming(false);
  }

  if (loading) {
    return <p className="text-gray-500 py-8 text-center">Loading...</p>;
  }

  if (error && !rows.length) {
    return <p className="text-red-500 py-8 text-center">{error}</p>;
  }

  const readyCount = rows.filter((r) => r.status === "ready").length;
  const needsAttentionCount = rows.filter(
    (r) => r.status === "needs_attention",
  ).length;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">Review Import</h1>
          <p className="text-gray-500 text-sm mt-1">{filename}</p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={() => router.push("/import")}
            className="px-4 py-2 text-sm text-gray-600 border rounded-md hover:bg-gray-50"
          >
            Cancel
          </button>
          {!confirmResult && (
            <button
              onClick={handleConfirm}
              disabled={confirming || readyCount === 0}
              className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {confirming
                ? "Importing..."
                : `Import ${readyCount} Transaction${readyCount !== 1 ? "s" : ""}`}
            </button>
          )}
        </div>
      </div>

      {needsAttentionCount > 0 && !confirmResult && (
        <div className="mb-4 p-3 rounded-md text-sm bg-amber-50 text-amber-800 border border-amber-200">
          {needsAttentionCount} row{needsAttentionCount !== 1 ? "s" : ""} need
          attention. Edit to resolve warnings before importing.
        </div>
      )}

      {confirmResult && (
        <div className="mb-4 p-4 rounded-md text-sm bg-green-50 text-green-800 border border-green-200">
          <p>
            Import complete: {confirmResult.inserted} inserted,{" "}
            {confirmResult.duplicates} duplicates skipped.
          </p>
          <button
            onClick={() => router.push("/transactions")}
            className="mt-2 text-blue-600 hover:underline text-sm"
          >
            View Transactions
          </button>
        </div>
      )}

      {error && (
        <div className="mb-4 p-4 rounded-md text-sm bg-red-50 text-red-800 border border-red-200">
          <p>{error}</p>
        </div>
      )}

      <ReviewTable
        importId={importId}
        rows={rows}
        onRowUpdated={handleRowUpdated}
      />
    </div>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/app/import/page.tsx frontend/app/import/\[importId\]/page.tsx
git commit -m "feat(frontend): add file upload and review page for imports"
```

---

## Slice 4 — End-to-end Verification

### Task 17: Rebuild and test the full flow

- [ ] **Step 1: Add ANTHROPIC_API_KEY to .env**

Add your Anthropic API key to `src/.env`:

```
ANTHROPIC_API_KEY=sk-ant-...your-key...
```

- [ ] **Step 2: Rebuild the stack**

Run: `docker-compose down && docker-compose up --build -d`
Expected: All three services start. Backend logs show "Migrations complete" (migration 3 applied).

- [ ] **Step 3: Verify migration applied**

Run: `docker-compose logs backend --tail 5`
Expected: "Migrations complete" — the `imports` and `import_staging_rows` tables now exist.

- [ ] **Step 4: Test file upload via curl**

Create a test CSV file and upload it:

```bash
cat > /tmp/test_import.csv << 'EOF'
Run Date,Action,Symbol,Quantity,Price,Amount
03/15/2024,YOU BOUGHT,AAPL,10,172.50,-1725.00
04/01/2024,DIVIDEND,AAPL,,0.24,2.40
EOF

curl -s -X POST http://localhost:8080/api/imports/upload \
  -F "file=@/tmp/test_import.csv" | python3 -m json.tool
```

Expected: JSON response with `import.id`, `import.status: "extracted"`, and `rows` array with 1 transaction (AAPL buy) and the dividend filtered out.

- [ ] **Step 5: Test in browser**

1. Open `http://localhost:3000/import`
2. Verify both Alpaca and File Import sections render
3. Upload the test CSV
4. Verify redirect to review page with extracted rows
5. Click "Import N Transactions"
6. Verify success toast shows inserted count
7. Navigate to Transactions page and verify rows appear

- [ ] **Step 6: Test idempotency**

Re-upload the same CSV. Confirm the response shows 0 inserted, N duplicates.

- [ ] **Step 7: Commit any fixes**

```bash
git add -A
git commit -m "fix: address integration issues from end-to-end testing"
```

---

## Summary of API routes added

| Method  | Route                                  | Purpose                                           |
| ------- | -------------------------------------- | ------------------------------------------------- |
| `POST`  | `/api/imports/upload`                  | Upload file, extract via LLM, return staging rows |
| `GET`   | `/api/imports/{importId}`              | Get import detail with staging rows               |
| `PATCH` | `/api/imports/{importId}/rows/{rowId}` | Edit a staging row                                |
| `POST`  | `/api/imports/{importId}/confirm`      | Commit approved rows to `transactions`            |
| `POST`  | `/extract` (data-service)              | LLM extraction endpoint                           |
