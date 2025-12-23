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
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "org not found"})
		return
	}
	sub, _ := s.Store.GetSubscription(r.Context(), org.ID)

	writeJSON(w, http.StatusOK, map[string]any{
		"userId": userID,
		"org":    org,
		"sub":    sub,
	})
}

func (s *Server) handleClusters(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	org, err := s.Store.GetOwnerOrg(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "org not found"})
		return
	}

	switch r.Method {

	case http.MethodGet:
		list, err := s.Store.ListClusters(r.Context(), org.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"clusters": list})

	case http.MethodPost:
		sub, _ := s.Store.GetSubscription(r.Context(), org.ID)
		max, _ := s.Store.PlanMaxClusters(r.Context(), sub.PlanID)
		cnt, _ := s.Store.CountClusters(r.Context(), org.ID)
		if cnt >= max {
			writeJSON(w, http.StatusPaymentRequired, map[string]any{
				"error": "plan limit reached (upgrade required)",
			})
			return
		}

		var req struct {
			Name  string `json:"name"`
			Notes string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad json"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name required"})
			return
		}

		c, err := s.Store.CreateCluster(r.Context(), org.ID, req.Name, req.Notes)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, c)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleScans(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	org, err := s.Store.GetOwnerOrg(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "org not found"})
		return
	}

	switch r.Method {

	case http.MethodGet:
		clusterID := strings.TrimSpace(r.URL.Query().Get("clusterId"))
		if clusterID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "clusterId required"})
			return
		}
		list, err := s.Store.ListScans(r.Context(), org.ID, clusterID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scans": list})

	case http.MethodPost:
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		clusterID := strings.TrimSpace(r.FormValue("clusterId"))
		if clusterID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "clusterId required"})
			return
		}

		file, _, err := r.FormFile("rbac")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "rbac file required"})
			return
		}
		defer file.Close()

		content, err := io.ReadAll(io.LimitReader(file, 64<<20))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "read failed"})
			return
		}

		data, err := loader.LoadFromBytes(content)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		perms := rbac.BuildSubjectPermissions(
			data.Roles,
			data.ClusterRoles,
			data.RoleBindings,
			data.ClusterRoleBindings,
		)

		sum := BuildSummary(perms)
		full := BuildFullReport(perms)

		sc, err := s.Store.CreateScan(r.Context(), org.ID, clusterID, "upload")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		if err := s.Store.UpsertScanResult(r.Context(), sc.ID, sum, full); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"scan":    sc,
			"summary": sum,
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleScanReport(w http.ResponseWriter, r *http.Request) {
	scanID := strings.TrimSpace(r.URL.Query().Get("scanId"))
	if scanID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "scanId required"})
		return
	}
	sum, full, err := s.Store.GetScanReport(r.Context(), scanID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "report not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": sum,
		"report":  full,
		"ts":      time.Now().UTC(),
	})
}
