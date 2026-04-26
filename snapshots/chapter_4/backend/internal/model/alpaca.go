package model

type AlpacaOrder struct {
	OrderID        string  `json:"order_id"`
	Ticker         string  `json:"ticker"`
	Side           string  `json:"side"`
	Qty            float64 `json:"qty"`
	FilledAvgPrice float64 `json:"filled_avg_price"`
	FilledAt       string  `json:"filled_at"`
}

type ImportResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Total   int `json:"total"`
}
