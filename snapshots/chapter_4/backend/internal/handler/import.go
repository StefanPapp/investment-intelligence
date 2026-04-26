package handler

import (
	"encoding/json"
	"net/http"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type ImportServiceInterface interface {
	ImportAlpacaOrders() (*model.ImportResult, error)
}

type ImportHandler struct {
	Svc ImportServiceInterface
}

func (h *ImportHandler) ImportAlpaca(w http.ResponseWriter, r *http.Request) {
	result, err := h.Svc.ImportAlpacaOrders()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
