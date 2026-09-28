package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-filmmaker/api/internal/config"
	"github.com/ai-filmmaker/api/internal/handlers"
)

func TestServerRoutes(t *testing.T) {
	cfg := &config.Config{
		Port: "8080",
	}
	s := New(cfg, nil)

	t.Run("health check route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()

		s.Router().ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var resp handlers.HealthResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != "ok" || resp.Service != "api" {
			t.Errorf("unexpected health response: %+v", resp)
		}
	})

	t.Run("cors preflight options", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/projects", nil)
		rr := httptest.NewRecorder()

		s.Router().ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content for OPTIONS, got %d", rr.Code)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("expected CORS allow origin header")
		}
	})
}
