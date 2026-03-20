package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"coding-plan-mask/internal/config"
	"coding-plan-mask/internal/storage"

	"go.uber.org/zap"
)

type flushingRecorder struct {
	*httptest.ResponseRecorder
}

func (r *flushingRecorder) Flush() {}

func TestBuildHeadersPreservesRequestHeadersAndOverridesAuth(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DisguiseTool = "claudecode"

	p := &Proxy{cfg: cfg}
	provider := &config.ProviderConfig{
		AuthHeader:   "Authorization",
		AuthPrefix:   "Bearer ",
		ExtraHeaders: map[string]string{},
	}

	requestHeaders := http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer local-key"},
		"X-Custom":      []string{"custom-value"},
	}

	headers := p.buildHeaders(provider, "test-key", requestHeaders)

	if got := headers.Get("Accept"); got != "application/json" {
		t.Fatalf("expected Accept header to be preserved, got %q", got)
	}
	if got := headers.Get("Authorization"); got != "Bearer test-key" {
		t.Fatalf("expected upstream Authorization header, got %q", got)
	}
	if got := headers.Get("User-Agent"); got != cfg.GetEffectiveUserAgent() {
		t.Fatalf("expected disguised User-Agent, got %q", got)
	}
	if got := headers.Get("X-Custom"); got != "custom-value" {
		t.Fatalf("expected custom header to be preserved, got %q", got)
	}
}

func TestBuildTargetURLPreservesPathAndQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/chat/completions?foo=bar", nil)

	got := buildTargetURL("https://example.com/api/coding/paas/v4", req)
	want := "https://example.com/api/coding/paas/v4/chat/completions?foo=bar"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestHandleStreamResponsePreservesEventBoundaries(t *testing.T) {
	cfg := config.DefaultConfig()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer store.Close()

	p := New(cfg, zap.NewNop(), store)

	recorder := &flushingRecorder{ResponseRecorder: httptest.NewRecorder()}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(strings.NewReader("data: {\"usage\":{\"completion_tokens\":3}}\n\ndata: [DONE]\n\n")),
	}

	p.handleStreamResponseWithStats(recorder, resp, time.Now(), http.MethodPost, "/chat/completions", "glm-4-flash", "127.0.0.1", 2, "{}")

	body := recorder.Body.String()
	if !strings.Contains(body, "\n\n") {
		t.Fatalf("expected SSE event boundary in body, got %q", body)
	}

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 status code, got %d", recorder.Code)
	}
}
