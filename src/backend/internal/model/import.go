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
