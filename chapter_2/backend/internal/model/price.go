package model

import "time"

type PriceCache struct {
	Ticker    string    `json:"ticker"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	FetchedAt time.Time `json:"fetched_at"`
}
