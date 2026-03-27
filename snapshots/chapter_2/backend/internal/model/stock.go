package model

import (
	"time"

	"github.com/google/uuid"
)

type Stock struct {
	ID        uuid.UUID `json:"id"`
	Ticker    string    `json:"ticker"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
