package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDaemonAuth_MissingAuth(t *testing.T) {
	mw := DaemonAuth(nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	}))
	req := httptest.NewRequest("POST", "/api/daemon/heartbeat", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDaemonAuth_InvalidMDT_NilQueries(t *testing.T) {
	mw := DaemonAuth(nil) // no DB
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	}))
	req := httptest.NewRequest("POST", "/api/daemon/heartbeat", nil)
	req.Header.Set("Authorization", "Bearer mdt_unknown")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDaemonAuth_UnsupportedTokenType(t *testing.T) {
	mw := DaemonAuth(nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	}))
	req := httptest.NewRequest("POST", "/api/daemon/heartbeat", nil)
	req.Header.Set("Authorization", "Bearer mul_some_pat_token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-mdt_ token, got %d", w.Code)
	}
}

// TestDaemonOrUserAuth_LoopbackPassesAsSingleton: on a personal/loopback
// deployment a daemon with no provisioned mdt_ token authenticates as the
// singleton user, so registration and claim work without daemon-token
// provisioning. This is the path that restores daemon connectivity on the fork.
func TestDaemonOrUserAuth_LoopbackPassesAsSingleton(t *testing.T) {
	mw := DaemonOrUserAuth(nil, nil)
	var gotUser string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("POST", "/api/daemon/register", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for loopback, got %d: %s", w.Code, w.Body.String())
	}
	if gotUser != SingletonUserID {
		t.Fatalf("expected X-User-ID=%s, got %q", SingletonUserID, gotUser)
	}
}

// TestDaemonOrUserAuth_MDTTokenStillValidated: presenting an mdt_ token still
// routes to daemon-token validation even from loopback, so a bad daemon token
// is rejected rather than silently downgraded to singleton auth.
func TestDaemonOrUserAuth_MDTTokenStillValidated(t *testing.T) {
	mw := DaemonOrUserAuth(nil, nil) // nil queries → mdt_ lookup fails closed
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called for an invalid mdt_ token")
	}))
	req := httptest.NewRequest("POST", "/api/daemon/register", nil)
	req.Header.Set("Authorization", "Bearer mdt_unknown")
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid mdt_ token, got %d", w.Code)
	}
}

// TestDaemonOrUserAuth_NonLoopbackNoTokenRejected: a non-loopback client with
// no token is still rejected — the loopback bypass does not widen to the network.
func TestDaemonOrUserAuth_NonLoopbackNoTokenRejected(t *testing.T) {
	mw := DaemonOrUserAuth(nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not be called")
	}))
	req := httptest.NewRequest("POST", "/api/daemon/register", nil)
	req.RemoteAddr = "10.0.0.5:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-loopback without token, got %d", w.Code)
	}
}
