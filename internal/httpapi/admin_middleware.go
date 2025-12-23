package httpapi

import "net/http"

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims.Sub == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing auth token"})
			return
		}
		if !claims.Admin {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "admin only"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
