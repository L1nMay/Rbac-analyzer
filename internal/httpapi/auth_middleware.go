package httpapi

import (
	"context"
	"net/http"
	"strings"

	"rbac-analyzer/internal/security"
)

type ctxKey string

const (
	ctxUserID  ctxKey = "user_id"
	ctxClaims  ctxKey = "claims"
	bearerPref        = "bearer "
)

func AuthMiddleware(jwtKey []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if auth == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing auth token"})
			return
		}

		low := strings.ToLower(auth)
		if strings.HasPrefix(low, bearerPref) {
			auth = strings.TrimSpace(auth[len(bearerPref):])
		}

		if auth == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing auth token"})
			return
		}

		claims, err := security.VerifyJWT(jwtKey, auth)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid token"})
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, claims.Sub)
		ctx = context.WithValue(ctx, ctxClaims, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(r *http.Request) string {
	v := r.Context().Value(ctxUserID)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func GetClaims(r *http.Request) security.Claims {
	v := r.Context().Value(ctxClaims)
	if v == nil {
		return security.Claims{}
	}
	c, _ := v.(security.Claims)
	return c
}
