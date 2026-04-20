package seed

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

// Position describes a single holding to seed.
type Position struct {
	Ticker        string
	Name          string
	Shares        string // string for NUMERIC precision (supports fractional shares)
	PricePerShare string // stored as string to avoid float; parsed by DB as NUMERIC
}

// ReferencePortfolio groups positions under a label (for logging only).
type ReferencePortfolio struct {
	Label     string
	Positions []Position
}

// ReferencePortfolios returns the four chapter-4 reference portfolios with
// exact share counts and prices from the book.
func ReferencePortfolios() []ReferencePortfolio {
	return []ReferencePortfolio{
		{
			Label: "AI & Machine Learning",
			Positions: []Position{
				{Ticker: "AMZN", Name: "Amazon.com Inc.", Shares: "13", PricePerShare: "250.56"},
				{Ticker: "GOOGL", Name: "Alphabet Inc.", Shares: "10", PricePerShare: "336.02"},
				{Ticker: "NVDA", Name: "NVIDIA Corporation", Shares: "17", PricePerShare: "198.35"},
			},
		},
		{
			Label: "Automotive & EV",
			Positions: []Position{
				{Ticker: "OUST", Name: "Ouster Inc.", Shares: "138", PricePerShare: "24.17"},
				{Ticker: "AEVA", Name: "Aeva Technologies Inc.", Shares: "226", PricePerShare: "14.76"},
				{Ticker: "BYDDY", Name: "BYD Company Limited", Shares: "232", PricePerShare: "14.35"},
			},
		},
		{
			Label: "Quantum Computing",
			Positions: []Position{
				{Ticker: "QBTS", Name: "D-Wave Quantum Inc.", Shares: "155", PricePerShare: "21.52"},
				{Ticker: "QUBT", Name: "Quantum Computing Inc.", Shares: "353", PricePerShare: "9.44"},
				{Ticker: "RGTI", Name: "Rigetti Computing Inc.", Shares: "171", PricePerShare: "19.45"},
			},
		},
		{
			Label: "Crypto & Blockchain",
			Positions: []Position{
				{Ticker: "BTC-USD", Name: "Bitcoin USD", Shares: "0.044", PricePerShare: "75830.00"},
				{Ticker: "ETH-USD", Name: "Ethereum USD", Shares: "1.415", PricePerShare: "2355.17"},
				{Ticker: "SOL-USD", Name: "Solana USD", Shares: "37.55", PricePerShare: "88.78"},
			},
		},
	}
}

// ReseedDatabase deletes all data and reseeds from scratch.
// Only call this on the test database.
func ReseedDatabase(db *sql.DB) error {
	log.Println("RESEED_TEST_DB=true — clearing test data")
	for _, table := range []string{"transactions", "prices_cache", "stocks"} {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	log.Println("Test data cleared — reseeding")
	return insertReferenceData(db)
}

// SeedIfEmpty checks whether the stocks table is empty. If it is, it inserts
// all reference portfolios as buy transactions dated 2026-04-17.
func SeedIfEmpty(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM stocks").Scan(&count); err != nil {
		return fmt.Errorf("count stocks: %w", err)
	}
	if count > 0 {
		log.Printf("Stocks table has %d rows — skipping seed", count)
		return nil
	}

	log.Println("Stocks table is empty — seeding reference data")
	return insertReferenceData(db)
}

func insertReferenceData(db *sql.DB) error {
	stockRepo := &repository.StockRepo{DB: db}

	const txnDate = "2026-04-17"

	for _, rp := range ReferencePortfolios() {
		log.Printf("  Seeding: %s", rp.Label)
		for _, pos := range rp.Positions {
			stock, err := stockRepo.GetOrCreate(pos.Ticker, pos.Name)
			if err != nil {
				return fmt.Errorf("seed stock %s: %w", pos.Ticker, err)
			}

			_, err = db.Exec(
				`INSERT INTO transactions (stock_id, transaction_type, shares, price_per_share, transaction_date)
				 VALUES ($1, $2, $3, $4, $5)`,
				stock.ID, "buy", pos.Shares, pos.PricePerShare, txnDate,
			)
			if err != nil {
				return fmt.Errorf("seed transaction %s: %w", pos.Ticker, err)
			}
		}
	}

	log.Println("Seeding complete")
	return nil
}
