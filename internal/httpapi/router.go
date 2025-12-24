package httpapi

import (
	"net/http"

	"rbac-analyzer/internal/config"
	"rbac-analyzer/internal/store"
)

type Server struct {
	Cfg   config.Config
	Store *store.Store
	Web   http.Handler // static web
}

func NewServer(cfg config.Config, st *store.Store, web http.Handler) *Server {
	return &Server{Cfg: cfg, Store: st, Web: web}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public API
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/auth/register", s.handleRegister)
	mux.HandleFunc("/api/auth/login", s.handleLogin)

	// Billing hooks (stub)
	mux.HandleFunc("/api/billing/stripe/webhook", s.handleStripeWebhook)

	// Auth middleware key
	jwtKey := []byte(s.Cfg.JWTSecret)

	// App API (auth required)
	mux.Handle("/api/app/me", AuthMiddleware(jwtKey, http.HandlerFunc(s.handleMe)))
	mux.Handle("/api/app/clusters", AuthMiddleware(jwtKey, http.HandlerFunc(s.handleClusters)))
	mux.Handle("/api/app/scans", AuthMiddleware(jwtKey, http.HandlerFunc(s.handleScans)))
	mux.Handle("/api/app/scans/diff", AuthMiddleware(jwtKey, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleDiffScans(w, r)
	})))
	mux.Handle("/api/app/scan/report", AuthMiddleware(jwtKey, http.HandlerFunc(s.handleScanReport)))

	// Admin API (auth + admin required)

	// список пользователей
	mux.Handle(
		"/api/admin/users",
		AuthMiddleware(
			jwtKey,
			RequireAdmin(http.HandlerFunc(s.handleAdminUsers)),
		),
	)

	// toggle admin (POST /api/admin/users/{id}/toggle-admin)
	mux.Handle(
		"/api/admin/users/",
		AuthMiddleware(
			jwtKey,
			RequireAdmin(http.HandlerFunc(s.handleAdminToggleUser)),
		),
	)

	// список организаций
	mux.Handle(
		"/api/admin/orgs",
		AuthMiddleware(
			jwtKey,
			RequireAdmin(http.HandlerFunc(s.handleAdminOrgs)),
		),
	)
	mux.Handle(
		"/api/admin/audit",
		AuthMiddleware(jwtKey, RequireAdmin(http.HandlerFunc(s.handleAdminAudit))),
	)

	// Static site last
	mux.Handle("/", s.Web)

	return withHeaders(mux)
}

func withHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
