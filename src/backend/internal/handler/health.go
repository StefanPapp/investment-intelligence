package handler

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	DbTarget string `json:"db_target"`
}

var dbTarget string

// SetDbTarget stores the database target (prod/test) for the health endpoint.
func SetDbTarget(target string) {
	dbTarget = target
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse{
		Status:   "ok",
		Service:  "backend",
		DbTarget: dbTarget,
	})
}
