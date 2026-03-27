package repository

import (
	"database/sql"
	"fmt"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type PortfolioRepo struct {
	DB *sql.DB
}

func (r *PortfolioRepo) GetHoldings() ([]model.Holding, error) {
	rows, err := r.DB.Query(`
		SELECT s.ticker, s.name,
		       SUM(CASE WHEN t.transaction_type = 'buy' THEN t.shares ELSE -t.shares END) AS total_shares,
		       SUM(CASE WHEN t.transaction_type = 'buy' THEN t.shares * t.price_per_share ELSE 0 END) /
		       NULLIF(SUM(CASE WHEN t.transaction_type = 'buy' THEN t.shares ELSE 0 END), 0) AS avg_cost
		FROM stocks s
		JOIN transactions t ON t.stock_id = s.id
		GROUP BY s.id, s.ticker, s.name
		HAVING SUM(CASE WHEN t.transaction_type = 'buy' THEN t.shares ELSE -t.shares END) > 0
		ORDER BY s.ticker
	`)
	if err != nil {
		return nil, fmt.Errorf("get holdings: %w", err)
	}
	defer rows.Close()

	var holdings []model.Holding
	for rows.Next() {
		var h model.Holding
		if err := rows.Scan(&h.Ticker, &h.Name, &h.TotalShares, &h.AvgCost); err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		holdings = append(holdings, h)
	}
	return holdings, rows.Err()
}
