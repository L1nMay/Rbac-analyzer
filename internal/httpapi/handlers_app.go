package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"rbac-analyzer/internal/loader"
	"rbac-analyzer/internal/rbac"
)

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	org, err := s.Store.GetOwnerOrg(r.Context(), userID)
	if err != nil {
		http.Error(w, "org not found: "+err.Error(), http.StatusBadRequest)
		return
	}
	sub, _ := s.Store.GetSubscription(r.Context(), org.ID)

	writeJSON(w, map[string]any{
		"userId": userID,
		"org":    org,
		"sub":    sub,
	})
}

func (s *Server) handleClusters(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	org, err := s.Store.GetOwnerOrg(r.Context(), userID)
	if err != nil {
		http.Error(w, "org not found", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		list, err := s.Store.ListClusters(r.Context(), org.ID)
		if err != nil {
			http.Error(w, "list clusters failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"clusters": list})
		return

	case http.MethodPost:
		// enforce plan limits
		sub, _ := s.Store.GetSubscription(r.Context(), org.ID)
		max, _ := s.Store.PlanMaxClusters(r.Context(), sub.PlanID)
		cnt, _ := s.Store.CountClusters(r.Context(), org.ID)
		if cnt >= max {
			http.Error(w, "plan limit reached (upgrade required)", http.StatusPaymentRequired)
			return
		}

		var req struct {
			Name  string `json:"name"`
			Notes string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}

		c, err := s.Store.CreateCluster(r.Context(), org.ID, req.Name, req.Notes)
		if err != nil {
			http.Error(w, "create cluster failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, c)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleScans(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	org, err := s.Store.GetOwnerOrg(r.Context(), userID)
	if err != nil {
		http.Error(w, "org not found", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		clusterID := strings.TrimSpace(r.URL.Query().Get("clusterId"))
		if clusterID == "" {
			http.Error(w, "clusterId required", http.StatusBadRequest)
			return
		}
		list, err := s.Store.ListScans(r.Context(), org.ID, clusterID)
		if err != nil {
			http.Error(w, "list scans failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"scans": list})
		return

	case http.MethodPost:
		// upload scan YAML
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "failed to parse multipart: "+err.Error(), http.StatusBadRequest)
			return
		}

		clusterID := strings.TrimSpace(r.FormValue("clusterId"))
		if clusterID == "" {
			http.Error(w, "clusterId required", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("rbac")
		if err != nil {
			http.Error(w, "file field 'rbac' required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		content, err := io.ReadAll(io.LimitReader(file, 64<<20))
		if err != nil {
			http.Error(w, "read failed", http.StatusBadRequest)
			return
		}

		data, err := loader.LoadFromBytes(content)
		if err != nil {
			http.Error(w, "parse yaml failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		subjectPerms := rbac.BuildSubjectPermissions(
			data.Roles, data.ClusterRoles, data.RoleBindings, data.ClusterRoleBindings,
		)

		// Сбор summary (продаваемая часть)
		sum := BuildSummary(subjectPerms)

		// полный report как JSON-структура
		full := BuildFullReport(subjectPerms)

		sc, err := s.Store.CreateScan(r.Context(), org.ID, clusterID, "upload")
		if err != nil {
			http.Error(w, "create scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.Store.UpsertScanResult(r.Context(), sc.ID, sum, full); err != nil {
			http.Error(w, "save scan result failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]any{
			"scan":    sc,
			"summary": sum,
		})
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleScanReport(w http.ResponseWriter, r *http.Request) {
	scanID := strings.TrimSpace(r.URL.Query().Get("scanId"))
	if scanID == "" {
		http.Error(w, "scanId required", http.StatusBadRequest)
		return
	}
	sum, full, err := s.Store.GetScanReport(r.Context(), scanID)
	if err != nil {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{
		"summary": sum,
		"report":  full,
		"ts":      time.Now().UTC(),
	})
}

// ---- Summary helpers (MVP-коммерческий смысл) ----

func BuildSummary(sp rbac.SubjectPermissions) map[string]any {
	type counts struct {
		Subjects    int `json:"subjects"`
		Roles       int `json:"roles"`
		Perms       int `json:"perms"`
		DangerRoles int `json:"dangerRoles"`
	}
	c := counts{}
	topDanger := make([]map[string]any, 0)

	for subj, roles := range sp {
		c.Subjects++
		dCount := 0
		pCount := 0
		for _, r := range roles {
			c.Roles++
			pCount += len(r.Permissions)
			if r.Dangerous {
				c.DangerRoles++
				dCount++
			}
		}
		c.Perms += pCount
		if dCount > 0 {
			topDanger = append(topDanger, map[string]any{
				"subject":     subj.String(),
				"dangerRoles": dCount,
				"perms":       pCount,
			})
		}
	}

	// риск-скор: очень простой MVP
	riskScore := 0.0
	if c.Roles > 0 {
		riskScore = float64(c.DangerRoles) / float64(c.Roles) * 10.0
		if riskScore > 10 {
			riskScore = 10
		}
	}

	return map[string]any{
		"counts":    c,
		"riskScore": riskScore,
		"topDanger": topDanger,
	}
}

func BuildFullReport(sp rbac.SubjectPermissions) map[string]any {
	type subjOut struct {
		Subject string               `json:"subject"`
		Roles   []rbac.EffectiveRole `json:"roles"`
	}
	out := make([]subjOut, 0, len(sp))
	for sref, roles := range sp {
		out = append(out, subjOut{
			Subject: sref.String(),
			Roles:   roles,
		})
	}
	return map[string]any{"subjects": out}
}
