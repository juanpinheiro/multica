package middleware

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
)

// SingletonUserID is the implicit user injected by LoopbackAuth for all
// loopback-originated and correctly-tokened requests on personal deployments.
const SingletonUserID = "00000000-0000-0000-0000-000000000001"

// loopbackRanges are the IPv4 and IPv6 loopback prefixes. Connections from
// these addresses are trusted unconditionally — no token required.
var loopbackRanges = []netip.Prefix{
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("::1/128"),
}

// LoopbackAuth returns a middleware that:
//   - Lets loopback clients (127.0.0.0/8, ::1) through unconditionally,
//     injecting X-User-ID = SingletonUserID.
//   - When the peer is in trustedProxies, uses the leftmost X-Forwarded-For
//     address as the effective client IP for loopback detection.
//   - Rejects non-loopback clients unless MULTICA_TOKEN is set and the request
//     carries a matching Authorization: Bearer <token> header.
func LoopbackAuth(trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolveClientIP(r, trustedProxies)

			if isLoopback(ip) {
				r.Header.Set("X-User-ID", SingletonUserID)
				next.ServeHTTP(w, r)
				return
			}

			// Non-loopback path: require MULTICA_TOKEN + Bearer match.
			token := strings.TrimSpace(os.Getenv("MULTICA_TOKEN"))
			if token == "" {
				writeError(w, http.StatusUnauthorized, "server is not configured for non-loopback access")
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				writeError(w, http.StatusUnauthorized, "missing authorization")
				return
			}

			supplied := authHeader[len(prefix):]
			if subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			r.Header.Set("X-User-ID", SingletonUserID)
			next.ServeHTTP(w, r)
		})
	}
}

// ResolveClientIP is the exported variant of resolveClientIP for callers
// outside the middleware package (e.g. realtime.HandleWebSocket). Same
// semantics: trusted proxies → leftmost X-Forwarded-For, else r.RemoteAddr.
func ResolveClientIP(r *http.Request, trustedProxies []netip.Prefix) netip.Addr {
	return resolveClientIP(r, trustedProxies)
}

// IsLoopback is the exported variant of isLoopback. Used by realtime to
// extend the personal-fork loopback bypass beyond HTTP middlewares.
func IsLoopback(ip netip.Addr) bool {
	return isLoopback(ip)
}

// resolveClientIP returns the effective client IP for the request. When the
// peer address is a trusted proxy, the leftmost address from X-Forwarded-For
// is used; otherwise r.RemoteAddr is used directly.
func resolveClientIP(r *http.Request, trustedProxies []netip.Prefix) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// r.RemoteAddr has no port (shouldn't happen in production, but be
		// defensive).
		host = r.RemoteAddr
	}

	peerAddr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}

	if inPrefixList(peerAddr, trustedProxies) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Leftmost entry is the original client (right-most is closest
			// to us but may be forged; leftmost is appended by each hop).
			parts := strings.SplitN(xff, ",", 2)
			if candidate, err := netip.ParseAddr(strings.TrimSpace(parts[0])); err == nil {
				return candidate
			}
		}
	}

	return peerAddr
}

// inPrefixList reports whether ip falls within any of the given netip.Prefix entries.
func inPrefixList(ip netip.Addr, prefixes []netip.Prefix) bool {
	for _, p := range prefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// isLoopback reports whether ip is a loopback address (127.0.0.0/8 or ::1).
func isLoopback(ip netip.Addr) bool {
	return inPrefixList(ip, loopbackRanges)
}
