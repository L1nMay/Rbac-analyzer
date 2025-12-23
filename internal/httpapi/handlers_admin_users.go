package httpapi

import (
	"net/http"
	"strings"
)

func (s *Server) handleAdminToggleUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// /api/admin/users/{id}/toggle-admin
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	userID := parts[3]
	if userID == "" {
		http.Error(w, "user id required", http.StatusBadRequest)
		return
	}

	if err := s.Store.ToggleAdmin(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
