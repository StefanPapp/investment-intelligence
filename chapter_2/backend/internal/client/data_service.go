package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

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
