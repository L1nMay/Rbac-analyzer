package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"rbac-analyzer/internal/config"
	"rbac-analyzer/internal/db"
	"rbac-analyzer/internal/httpapi"
	"rbac-analyzer/internal/store"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	st := store.New(pool)

	// Static web handler (marketing + app) из embed FS
	web := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWeb(w, r)
	})

	srv := httpapi.NewServer(cfg, st, web)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Printf("RBAC Server listening on %s\n", cfg.Addr)
	if err := httpSrv.ListenAndServe(); err != nil {
		panic(err)
	}
}

func serveWeb(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path

	// Маркетинговые маршруты
	switch p {
	case "/":
		p = "web/marketing/index.html"
	case "/pricing":
		p = "web/marketing/pricing.html"
	case "/contact":
		p = "web/marketing/contact.html"
	default:
		// App маршруты
		if strings.HasPrefix(p, "/app") {
			if p == "/app" || p == "/app/" {
				p = "web/app/index.html"
			} else {
				// /app/style.css -> web/app/style.css
				p = path.Clean(strings.TrimPrefix(p, "/"))
				p = "web/" + p
			}
		} else {
			// Статика (например: /marketing/site.css)
			// /marketing/site.css -> web/marketing/site.css
			p = path.Clean(strings.TrimPrefix(p, "/"))
			p = "web/" + p
		}
	}

	// Защита от path traversal + гарантия префикса
	if !strings.HasPrefix(p, "web/") {
		http.NotFound(w, r)
		return
	}

	f, err := webFS.Open(p)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	// content-type
	switch {
	case strings.HasSuffix(p, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(p, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(p, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(p, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	_, _ = io.Copy(w, f)
}
