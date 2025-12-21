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

	// Marketing + App
	mux.Handle("/", s.Web)

	// Public API
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/auth/register", s.handleRegister)
	mux.HandleFunc("/api/auth/login", s.handleLogin)

	// Billing hooks (заготовка)
	mux.HandleFunc("/api/billing/stripe/webhook", s.handleStripeWebhook)

	// App API (auth required)
	mux.Handle("/api/app/me", AuthMiddleware([]byte(s.Cfg.JWTSecret), http.HandlerFunc(s.handleMe)))
	mux.Handle("/api/app/clusters", AuthMiddleware([]byte(s.Cfg.JWTSecret), http.HandlerFunc(s.handleClusters)))
	mux.Handle("/api/app/scans", AuthMiddleware([]byte(s.Cfg.JWTSecret), http.HandlerFunc(s.handleScans)))
	mux.Handle("/api/app/scan/report", AuthMiddleware([]byte(s.Cfg.JWTSecret), http.HandlerFunc(s.handleScanReport)))

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
