package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 200
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		rows, err := s.Store.AdminListUsers(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": rows})
		return

	case http.MethodPatch:
		var req struct {
			UserID  string `json:"userId"`
			IsAdmin bool   `json:"isAdmin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad json"})
			return
		}
		if err := s.Store.AdminSetUserAdmin(r.Context(), req.UserID, req.IsAdmin); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminOrgs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 200
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		rows, err := s.Store.AdminListOrgs(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"orgs": rows})
		return

	case http.MethodPatch:
		var req struct {
			OrgID  string `json:"orgId"`
			PlanID string `json:"planId"` // free | pro | enterprise (или твои планы)
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrgID == "" || req.PlanID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad json"})
			return
		}
		if err := s.Store.AdminSetOrgPlan(r.Context(), req.OrgID, req.PlanID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
