package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type PriceCacheRepo struct {
	DB *sql.DB
}

func (r *PriceCacheRepo) Get(ticker string, maxAge time.Duration) (*model.PriceCache, error) {
	p := &model.PriceCache{}
	err := r.DB.QueryRow(
		`SELECT ticker, price, currency, fetched_at FROM prices_cache
		 WHERE ticker = $1 AND fetched_at > NOW() - $2::interval`,
		ticker, fmt.Sprintf("%d seconds", int(maxAge.Seconds())),
	).Scan(&p.Ticker, &p.Price, &p.Currency, &p.FetchedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PriceCacheRepo) Upsert(ticker string, price float64, currency string) error {
	_, err := r.DB.Exec(
		`INSERT INTO prices_cache (ticker, price, currency, fetched_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (ticker) DO UPDATE SET price = $2, currency = $3, fetched_at = NOW()`,
		ticker, price, currency,
	)
	if err != nil {
		return fmt.Errorf("upsert price cache: %w", err)
	}
	return nil
}
