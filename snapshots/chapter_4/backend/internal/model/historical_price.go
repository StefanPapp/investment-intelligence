package model

type HistoricalPrice struct {
	Date     string   `json:"date"`
	Open     *float64 `json:"open"`
	High     *float64 `json:"high"`
	Low      *float64 `json:"low"`
	Close    *float64 `json:"close"`
	AdjClose *float64 `json:"adj_close"`
	Volume   *float64 `json:"volume"`
}

type HistoricalPriceResponse struct {
	Ticker   string            `json:"ticker"`
	Currency string            `json:"currency"`
	Interval string            `json:"interval"`
	Prices   []HistoricalPrice `json:"prices"`
}
