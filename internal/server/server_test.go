package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"coding-plan-mask/internal/config"
	"coding-plan-mask/internal/storage"

	"go.uber.org/zap"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	cfg := config.DefaultConfig()
	return New(cfg, zap.NewNop(), store, "test")
}

func TestRootRouteStillServesLocalInfo(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.SetupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for root route, got %d", rec.Code)
	}
}

func TestArbitraryProxyRouteIsHandled(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.SetupRoutes()

	req := httptest.NewRequest(http.MethodPost, "/chat/completions", bytes.NewBufferString(`{"model":"glm-5"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("expected arbitrary route to be proxied instead of 404")
	}
}

func TestVersionedProxyRouteIsHandled(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.SetupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("expected versioned route to be proxied instead of 404")
	}
}
