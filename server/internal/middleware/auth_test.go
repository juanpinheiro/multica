package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth_MatTokenNilQueries_Returns401(t *testing.T) {
	mw := Auth(nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	}))

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer mat_some_task_token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for mat_ with nil queries, got %d", w.Code)
	}
}

// TestAuth_LoopbackNoAuth: loopback request with no auth passes via loopback fallback.
func TestAuth_LoopbackNoAuth(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "")
	var gotUserID string
	mw := Auth(nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for loopback, got %d: %s", w.Code, w.Body.String())
	}
	if gotUserID != SingletonUserID {
		t.Fatalf("expected X-User-ID=%s, got %q", SingletonUserID, gotUserID)
	}
}

// TestAuth_StripsClientSuppliedActorSource: X-Actor-Source supplied by client is stripped.
func TestAuth_StripsClientSuppliedActorSource(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "")
	var gotActorSource string
	mw := Auth(nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotActorSource = r.Header.Get("X-Actor-Source")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("X-Actor-Source", "task_token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for loopback, got %d: %s", w.Code, w.Body.String())
	}
	if gotActorSource != "" {
		t.Fatalf("X-Actor-Source must be cleared, got %q", gotActorSource)
	}
}

// TestAuth_NonLoopbackNoToken: non-loopback without MULTICA_TOKEN → 401.
func TestAuth_NonLoopbackNoToken(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "")
	mw := Auth(nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	}))

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "10.0.0.5:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-loopback without token, got %d", w.Code)
	}
}
