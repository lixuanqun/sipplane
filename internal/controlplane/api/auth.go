package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// BearerAuth wraps next so that /v1/* requires Authorization: Bearer <token>
// when token is non-empty. /healthz stays anonymous for probes.
func BearerAuth(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	want := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		got := bearerToken(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="sipplane-control"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(h string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
}
