package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

// DataServiceError represents an error response from the data service with the upstream status code.
type DataServiceError struct {
	StatusCode int
	Message    string
}

func (e *DataServiceError) Error() string {
	return e.Message
}

type DataServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewDataServiceClient(baseURL string) *DataServiceClient {
	return &DataServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *DataServiceClient) GetPrice(ticker string) (*model.PriceCache, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/price/%s", c.baseURL, ticker))
	if err != nil {
		return nil, fmt.Errorf("fetch price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("data service returned %d for ticker %s", resp.StatusCode, ticker)
	}

	var result struct {
		Ticker    string  `json:"ticker"`
		Price     float64 `json:"price"`
		Currency  string  `json:"currency"`
		FetchedAt string  `json:"fetched_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	fetchedAt, _ := time.Parse(time.RFC3339, result.FetchedAt)
	return &model.PriceCache{
		Ticker:    result.Ticker,
		Price:     result.Price,
		Currency:  result.Currency,
		FetchedAt: fetchedAt,
	}, nil
}

func (c *DataServiceClient) GetAlpacaOrders() ([]model.AlpacaOrder, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/alpaca/orders", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("fetch alpaca orders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &DataServiceError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("data service returned %d for alpaca orders", resp.StatusCode),
		}
	}

	var orders []model.AlpacaOrder
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, fmt.Errorf("decode alpaca orders: %w", err)
	}
	return orders, nil
}

func (c *DataServiceClient) GetPriceHistory(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
	url := fmt.Sprintf("%s/price/%s/history?start=%s&end=%s", c.baseURL, ticker, start, end)

	// Use a longer timeout for potentially large historical data requests
	historyClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := historyClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch price history: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &DataServiceError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("data service returned %d for ticker %s history", resp.StatusCode, ticker),
		}
	}

	var result model.HistoricalPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode history response: %w", err)
	}

	return &result, nil
}
