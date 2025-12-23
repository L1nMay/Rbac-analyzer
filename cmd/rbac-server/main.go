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

	// Static web handler (marketing + auth + app) из embed FS
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

	// favicon
	if p == "/favicon.ico" {
		serveEmbeddedFile(w, r, "web/favicon.svg")
		return
	}

	// ---------- ADMIN ----------
	if p == "/admin" || p == "/admin/" {
		serveEmbeddedFile(w, r, "web/admin/login.html")
		return
	}

	if p == "/admin/dashboard" {
		serveEmbeddedFile(w, r, "web/admin/index.html")
		return
	}

	if strings.HasPrefix(p, "/admin/") {
		p = path.Clean(strings.TrimPrefix(p, "/"))
		serveEmbeddedFile(w, r, "web/"+p)
		return
	}

	// ---------- AUTH ----------
	if p == "/login" {
		serveEmbeddedFile(w, r, "web/auth/login.html")
		return
	}
	if p == "/register" {
		serveEmbeddedFile(w, r, "web/auth/register.html")
		return
	}

	// ---------- MARKETING ----------
	switch p {
	case "/":
		p = "web/marketing/index.html"
	case "/pricing":
		p = "web/marketing/pricing.html"
	case "/contact":
		p = "web/marketing/contact.html"
	default:
		// ---------- APP ----------
		if strings.HasPrefix(p, "/app") {
			if p == "/app" || p == "/app/" {
				p = "web/app/index.html"
			} else {
				p = "web/" + path.Clean(strings.TrimPrefix(p, "/"))
			}
		} else {
			p = "web/" + path.Clean(strings.TrimPrefix(p, "/"))
		}
	}

	serveEmbeddedFile(w, r, p)
}

func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, p string) {
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

	case p == "/admin" || p == "/admin/":
		p = "web/admin/index.html"

	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	_, _ = io.Copy(w, f)
}
