package model

type Holding struct {
	Ticker       string  `json:"ticker"`
	Name         string  `json:"name"`
	TotalShares  float64 `json:"total_shares"`
	AvgCost      float64 `json:"avg_cost"`
	CurrentPrice float64 `json:"current_price"`
	MarketValue  float64 `json:"market_value"`
	GainLoss     float64 `json:"gain_loss"`
	GainLossPct  float64 `json:"gain_loss_pct"`
}

type Portfolio struct {
	Holdings      []Holding `json:"holdings"`
	TotalValue    float64   `json:"total_value"`
	TotalCost     float64   `json:"total_cost"`
	TotalGainLoss float64   `json:"total_gain_loss"`
}
