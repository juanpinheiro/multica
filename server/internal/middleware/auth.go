package middleware

import (
	"net/http"
	"net/netip"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func uuidToString(u pgtype.UUID) string { return util.UUIDToString(u) }

// Auth middleware handles:
//  1. Agent task tokens (mat_ prefix) — server-minted tokens for running agent processes.
//     Sets X-User-ID, X-Agent-ID, X-Task-ID, X-Workspace-ID, X-Actor-Source=task_token.
//  2. All other requests — delegated to LoopbackAuth, which accepts loopback connections
//     or correctly-tokened non-loopback connections.
func Auth(queries *db.Queries, trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	loopback := LoopbackAuth(trustedProxies)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// X-Actor-Source is server-set only — strip any client-supplied value.
			r.Header.Del("X-Actor-Source")

			tokenString, _ := extractToken(r)

			// Agent task token (mat_): minted at task-claim time for the agent process.
			if strings.HasPrefix(tokenString, "mat_") {
				if queries == nil {
					http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
					return
				}
				hash := auth.HashToken(tokenString)
				tt, err := queries.GetTaskTokenByHash(r.Context(), hash)
				if err != nil {
					http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
					return
				}
				r.Header.Set("X-User-ID", uuidToString(tt.UserID))
				r.Header.Set("X-Agent-ID", uuidToString(tt.AgentID))
				r.Header.Set("X-Task-ID", uuidToString(tt.TaskID))
				r.Header.Set("X-Workspace-ID", uuidToString(tt.WorkspaceID))
				r.Header.Set("X-Actor-Source", "task_token")
				next.ServeHTTP(w, r)
				return
			}

			// All other requests: loopback or MULTICA_TOKEN bearer.
			loopback(next).ServeHTTP(w, r)
		})
	}
}

// extractToken returns the bearer token and whether it came from a cookie.
// Priority: Authorization header > multica_auth cookie.
func extractToken(r *http.Request) (token string, fromCookie bool) {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString != authHeader {
			return tokenString, false
		}
	}

	if cookie, err := r.Cookie(auth.AuthCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, true
	}

	return "", false
}
