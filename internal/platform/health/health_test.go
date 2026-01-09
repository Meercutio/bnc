package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadiness(t *testing.T) {
	p := New()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	p.Readiness(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	p.SetReady(true)
	w = httptest.NewRecorder()
	p.Readiness(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
