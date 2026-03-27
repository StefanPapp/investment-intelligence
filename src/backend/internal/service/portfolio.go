package service

import (
	"log"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

const priceCacheTTL = 15 * time.Minute

type PortfolioService struct {
	PortfolioRepo  *repository.PortfolioRepo
	PriceCacheRepo *repository.PriceCacheRepo
	DataClient     *client.DataServiceClient
}

func (s *PortfolioService) GetPortfolio() (*model.Portfolio, error) {
	holdings, err := s.PortfolioRepo.GetHoldings()
	if err != nil {
		return nil, err
	}

	var totalValue, totalCost float64

	for i := range holdings {
		h := &holdings[i]
		price := s.fetchPrice(h.Ticker)
		h.CurrentPrice = price
		h.MarketValue = h.TotalShares * price
		h.GainLoss = h.MarketValue - (h.TotalShares * h.AvgCost)
		if h.AvgCost > 0 {
			h.GainLossPct = (h.CurrentPrice - h.AvgCost) / h.AvgCost * 100
		}
		totalValue += h.MarketValue
		totalCost += h.TotalShares * h.AvgCost
	}

	return &model.Portfolio{
		Holdings:      holdings,
		TotalValue:    totalValue,
		TotalCost:     totalCost,
		TotalGainLoss: totalValue - totalCost,
	}, nil
}

func (s *PortfolioService) GetPrice(ticker string) (*model.PriceCache, error) {
	// Check cache first
	cached, err := s.PriceCacheRepo.Get(ticker, priceCacheTTL)
	if err == nil {
		return cached, nil
	}

	// Fetch from python service
	price, err := s.DataClient.GetPrice(ticker)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if cacheErr := s.PriceCacheRepo.Upsert(ticker, price.Price, price.Currency); cacheErr != nil {
		log.Printf("WARNING: failed to cache price for %s: %v", ticker, cacheErr)
	}

	return price, nil
}

func (s *PortfolioService) fetchPrice(ticker string) float64 {
	price, err := s.GetPrice(ticker)
	if err != nil {
		log.Printf("WARNING: could not fetch price for %s: %v", ticker, err)
		return 0
	}
	return price.Price
}
