package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"rbac-analyzer/internal/loader"
	"rbac-analyzer/internal/rbac"
)

type analyzeResponse struct {
	Subjects []subjectResult `json:"subjects"`
}

type subjectResult struct {
	Subject string               `json:"subject"`
	Roles   []rbac.EffectiveRole `json:"roles"`
}

func main() {
	mux := http.NewServeMux()

	// UI: отдаём статические файлы из embed
	mux.HandleFunc("/", serveStatic)
	mux.HandleFunc("/api/analyze", handleAnalyze)

	addr := ":8080"
	fmt.Printf("RBAC server listening on http://0.0.0.0%s\n", addr)
	if err := http.ListenAndServe(addr, withSecurityHeaders(mux)); err != nil {
		panic(err)
	}
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// базовые заголовки безопасности
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if p == "/" {
		p = "/web/index.html"
	} else {
		// наши файлы лежат в embed с префиксом web/
		p = path.Clean("/web" + p)
		if !strings.HasPrefix(p, "/web/") {
			http.NotFound(w, r)
			return
		}
	}

	f, err := webFS.Open(strings.TrimPrefix(p, "/"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	// простой content-type
	switch {
	case strings.HasSuffix(p, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(p, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(p, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	_, _ = io.Copy(w, f)
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Параметры фильтрации
	dangerOnly := r.URL.Query().Get("dangerOnly") == "true"
	namespaceFilter := strings.TrimSpace(r.URL.Query().Get("namespace"))

	// multipart upload
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("rbac")
	if err != nil {
		http.Error(w, "missing file field 'rbac': "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Size == 0 {
		http.Error(w, "empty file", http.StatusBadRequest)
		return
	}

	// читаем в память
	content, err := io.ReadAll(io.LimitReader(file, 64<<20)) // 64MB limit
	if err != nil {
		http.Error(w, "failed to read file: "+err.Error(), http.StatusBadRequest)
		return
	}

	data, err := loader.LoadFromBytes(content)
	if err != nil {
		http.Error(w, "failed to parse yaml: "+err.Error(), http.StatusBadRequest)
		return
	}

	subjectPerms := rbac.BuildSubjectPermissions(
		data.Roles,
		data.ClusterRoles,
		data.RoleBindings,
		data.ClusterRoleBindings,
	)

	resp := analyzeResponse{
		Subjects: buildResponse(subjectPerms, dangerOnly, namespaceFilter),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}

func buildResponse(
	subjectPerms rbac.SubjectPermissions,
	dangerOnly bool,
	namespaceFilter string,
) []subjectResult {
	nsFilter := strings.TrimSpace(namespaceFilter)

	var out []subjectResult
	for subj, roles := range subjectPerms {
		filtered := filterRoles(roles, dangerOnly, nsFilter)
		if len(filtered) == 0 {
			continue
		}
		out = append(out, subjectResult{
			Subject: subj.String(),
			Roles:   filtered,
		})
	}
	return out
}

func filterRoles(roles []rbac.EffectiveRole, dangerOnly bool, nsFilter string) []rbac.EffectiveRole {
	if !dangerOnly && nsFilter == "" {
		return roles
	}
	var out []rbac.EffectiveRole
	for _, r := range roles {
		if dangerOnly && !r.Dangerous {
			continue
		}
		if nsFilter != "" {
			// роль namespace-ная — должна совпасть по SourceNamespace
			if !r.ClusterScope && r.SourceNamespace != nsFilter {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}
