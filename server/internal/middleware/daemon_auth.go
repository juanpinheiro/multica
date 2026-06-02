package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"

	"github.com/multica-ai/multica/server/internal/auth"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// Daemon context keys.
type daemonContextKey int

const (
	ctxKeyDaemonWorkspaceID daemonContextKey = iota
	ctxKeyDaemonID
	ctxKeyDaemonAuthPath
)

// Daemon auth path labels exposed via context for slow-log attribution.
const (
	DaemonAuthPathDaemonToken = "daemon_token"
)

// DaemonWorkspaceIDFromContext returns the workspace ID set by DaemonAuth middleware.
func DaemonWorkspaceIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyDaemonWorkspaceID).(string)
	return id
}

// DaemonIDFromContext returns the daemon ID set by DaemonAuth middleware.
func DaemonIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyDaemonID).(string)
	return id
}

// DaemonAuthPathFromContext returns which token kind authenticated this
// request — "daemon_token" — for telemetry. Empty when the request did not
// pass through DaemonAuth.
func DaemonAuthPathFromContext(ctx context.Context) string {
	p, _ := ctx.Value(ctxKeyDaemonAuthPath).(string)
	return p
}

// WithDaemonContext returns a new context with the daemon workspace ID and daemon ID set.
// This is used by tests to simulate daemon token authentication.
func WithDaemonContext(ctx context.Context, workspaceID, daemonID string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyDaemonWorkspaceID, workspaceID)
	ctx = context.WithValue(ctx, ctxKeyDaemonID, daemonID)
	ctx = context.WithValue(ctx, ctxKeyDaemonAuthPath, DaemonAuthPathDaemonToken)
	return ctx
}

// DaemonAuth validates daemon auth tokens (mdt_ prefix).
// Only mdt_ tokens are accepted; all other tokens are rejected.
func DaemonAuth(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				slog.Debug("daemon_auth: missing authorization header", "path", r.URL.Path)
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				slog.Debug("daemon_auth: invalid format", "path", r.URL.Path)
				writeError(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			// Daemon token: "mdt_" prefix.
			if strings.HasPrefix(tokenString, "mdt_") {
				if queries == nil {
					writeError(w, http.StatusUnauthorized, "invalid daemon token")
					return
				}
				hash := auth.HashToken(tokenString)
				dt, err := queries.GetDaemonTokenByHash(r.Context(), hash)
				if err != nil {
					slog.Warn("daemon_auth: invalid daemon token", "path", r.URL.Path, "error", err)
					writeError(w, http.StatusUnauthorized, "invalid daemon token")
					return
				}

				ctx := context.WithValue(r.Context(), ctxKeyDaemonWorkspaceID, uuidToString(dt.WorkspaceID))
				ctx = context.WithValue(ctx, ctxKeyDaemonID, dt.DaemonID)
				ctx = context.WithValue(ctx, ctxKeyDaemonAuthPath, DaemonAuthPathDaemonToken)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			slog.Warn("daemon_auth: unsupported token type", "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "invalid daemon token")
		})
	}
}

// DaemonOrUserAuth authenticates the /api/daemon routes by daemon token when one
// is presented, and otherwise falls back to standard user auth (loopback
// singleton on personal deployments, mat_ task tokens, or a MULTICA_TOKEN
// bearer for non-loopback clients).
//
// The daemon route handlers never require a daemon-token context: every access
// check (requireDaemonWorkspaceAccess / requireDaemonRuntimeAccess /
// requireDaemonTaskAccess) resolves the workspace from the runtime or task and
// falls back to a workspace-membership check when no daemon-token context is
// present. That makes member-authed callers first-class on the daemon API.
//
// This restores daemon connectivity on the single-user fork, where the mdt_
// daemon-token provisioning flow (previously tied to the multi-tenant
// "connect daemon" surface) was removed: a loopback daemon can now register and
// claim work as the singleton user without a provisioned daemon token, while a
// real mdt_ token still authenticates with its original workspace-scoped
// semantics.
func DaemonOrUserAuth(queries *db.Queries, trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	daemon := DaemonAuth(queries)
	user := Auth(queries, trustedProxies)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, _ := extractToken(r)
			if strings.HasPrefix(token, "mdt_") {
				daemon(next).ServeHTTP(w, r)
				return
			}
			user(next).ServeHTTP(w, r)
		})
	}
}
