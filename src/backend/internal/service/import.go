package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

type ImportService struct {
	StockRepo  *repository.StockRepo
	ImportRepo *repository.ImportRepo
	DataClient *client.DataServiceClient
}

func (s *ImportService) ImportAlpacaOrders() (*model.ImportResult, error) {
	orders, err := s.DataClient.GetAlpacaOrders()
	if err != nil {
		return nil, fmt.Errorf("fetch alpaca orders: %w", err)
	}

	result := &model.ImportResult{Total: len(orders)}

	for _, order := range orders {
		ticker := strings.ToUpper(order.Ticker)

		stock, err := s.StockRepo.GetOrCreate(ticker, ticker)
		if err != nil {
			log.Printf("WARNING: skip order %s — stock error: %v", order.OrderID, err)
			continue
		}

		// Parse filled_at to extract date (YYYY-MM-DD)
		txnDate := order.FilledAt
		if len(txnDate) >= 10 {
			txnDate = txnDate[:10]
		}

		created, err := s.ImportRepo.UpsertTransaction(
			stock.ID,
			order.Side,
			order.Qty,
			order.FilledAvgPrice,
			txnDate,
			"alpaca",
			order.OrderID,
		)
		if err != nil {
			log.Printf("WARNING: skip order %s — upsert error: %v", order.OrderID, err)
			continue
		}

		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	log.Printf("Alpaca import complete: %d created, %d updated, %d total orders",
		result.Created, result.Updated, result.Total)
	return result, nil
}
