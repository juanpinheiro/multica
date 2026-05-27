package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func loopbackHandler(trustedProxies []netip.Prefix) http.Handler {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		w.Header().Set("X-User-ID", userID)
		w.WriteHeader(http.StatusOK)
	})
	return LoopbackAuth(trustedProxies)(next)
}

// TestLoopbackAuth_LoopbackNoAuth: loopback without Authorization header passes.
func TestLoopbackAuth_LoopbackNoAuth(t *testing.T) {
	handler := loopbackHandler(nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-User-ID"); got != SingletonUserID {
		t.Fatalf("expected X-User-ID=%s, got %q", SingletonUserID, got)
	}
}

// TestLoopbackAuth_LoopbackWithAuth: loopback with Authorization header still passes (loopback always wins).
func TestLoopbackAuth_LoopbackWithAuth(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "secret-token")
	handler := loopbackHandler(nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for loopback regardless of auth header, got %d", w.Code)
	}
	if got := w.Header().Get("X-User-ID"); got != SingletonUserID {
		t.Fatalf("expected X-User-ID=%s, got %q", SingletonUserID, got)
	}
}

// TestLoopbackAuth_NonLoopbackNoToken: non-loopback without MULTICA_TOKEN → 401 "server is not configured".
func TestLoopbackAuth_NonLoopbackNoToken(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "")
	handler := loopbackHandler(nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "10.0.0.5:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	body := w.Body.String()
	if !contains401Body(body, "server is not configured for non-loopback access") {
		t.Fatalf("unexpected body: %s", body)
	}
}

// TestLoopbackAuth_NonLoopbackTokenSetNoHeader: MULTICA_TOKEN set but no Authorization header → 401 "missing authorization".
func TestLoopbackAuth_NonLoopbackTokenSetNoHeader(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "secret-token")
	handler := loopbackHandler(nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "10.0.0.5:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	body := w.Body.String()
	if !contains401Body(body, "missing authorization") {
		t.Fatalf("unexpected body: %s", body)
	}
}

// TestLoopbackAuth_NonLoopbackWrongToken: MULTICA_TOKEN set + wrong token → 401 "invalid token".
func TestLoopbackAuth_NonLoopbackWrongToken(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "secret-token")
	handler := loopbackHandler(nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "10.0.0.5:54321"
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	body := w.Body.String()
	if !contains401Body(body, "invalid token") {
		t.Fatalf("unexpected body: %s", body)
	}
}

// TestLoopbackAuth_NonLoopbackCorrectToken: correct Bearer token passes.
func TestLoopbackAuth_NonLoopbackCorrectToken(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "secret-token")
	handler := loopbackHandler(nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "10.0.0.5:54321"
	req.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-User-ID"); got != SingletonUserID {
		t.Fatalf("expected X-User-ID=%s, got %q", SingletonUserID, got)
	}
}

// TestLoopbackAuth_TrustedProxyLoopbackForward: trusted proxy with XFF pointing to loopback → passes.
func TestLoopbackAuth_TrustedProxyLoopbackForward(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "")
	trustedProxies := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	handler := loopbackHandler(trustedProxies)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "10.0.0.1:54321" // trusted proxy
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when XFF is loopback via trusted proxy, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-User-ID"); got != SingletonUserID {
		t.Fatalf("expected X-User-ID=%s, got %q", SingletonUserID, got)
	}
}

// TestLoopbackAuth_TrustedProxyNonLoopbackForward: trusted proxy with XFF pointing to non-loopback → 401.
func TestLoopbackAuth_TrustedProxyNonLoopbackForward(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "")
	trustedProxies := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	handler := loopbackHandler(trustedProxies)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.RemoteAddr = "10.0.0.1:54321" // trusted proxy
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when XFF is non-loopback, got %d", w.Code)
	}
}

// contains401Body reports whether the response body matches the expected
// error message in the {"error":"..."} JSON envelope written by writeError.
func contains401Body(body, want string) bool {
	expected := `{"error":"` + want + `"}`
	return strings.TrimSpace(body) == expected
}
