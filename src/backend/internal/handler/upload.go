package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/service"
)

var allowedExtensions = map[string]bool{
	".csv": true, ".png": true, ".jpg": true, ".jpeg": true, ".pdf": true,
}

const maxUploadSize = 10 << 20 // 10 MB

type UploadHandler struct {
	Svc *service.StagingService
}

// Upload handles multipart file uploads, validates the extension, stores the file,
// and immediately triggers extraction. Returns an ImportDetail JSON response.
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, `{"error":"request too large or not multipart"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"missing file field"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		http.Error(w, `{"error":"unsupported file type"}`, http.StatusBadRequest)
		return
	}

	// Strip the leading dot to use as the file type label (e.g. "csv", "pdf").
	fileType := strings.TrimPrefix(ext, ".")

	uploadResult, err := h.Svc.Upload(header.Filename, fileType, file)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	detail, err := h.Svc.Extract(uploadResult.ImportID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// GetImport returns the current state of an import and its staging rows.
func (h *UploadHandler) GetImport(w http.ResponseWriter, r *http.Request) {
	importIDStr := chi.URLParam(r, "importId")
	importID, err := uuid.Parse(importIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid importId"}`, http.StatusBadRequest)
		return
	}

	detail, err := h.Svc.GetImport(importID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// PatchRow applies partial updates to a staging row.
// All body fields are optional; only non-null values are applied.
func (h *UploadHandler) PatchRow(w http.ResponseWriter, r *http.Request) {
	rowIDStr := chi.URLParam(r, "rowId")
	rowID, err := uuid.Parse(rowIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid rowId"}`, http.StatusBadRequest)
		return
	}

	var body struct {
		TradeDate     *string  `json:"trade_date"`
		Symbol        *string  `json:"symbol"`
		Side          *string  `json:"side"`
		Quantity      *float64 `json:"quantity"`
		PricePerShare *float64 `json:"price_per_share"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.Svc.UpdateRow(rowID, body.TradeDate, body.Symbol, body.Side, body.Quantity, body.PricePerShare); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Confirm confirms an import, converting all "ready" staging rows into transactions.
func (h *UploadHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	importIDStr := chi.URLParam(r, "importId")
	importID, err := uuid.Parse(importIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid importId"}`, http.StatusBadRequest)
		return
	}

	result, err := h.Svc.Confirm(importID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
