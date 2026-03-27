package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
)

func TestHealthReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Fatal("expected non-empty body")
	}
}
