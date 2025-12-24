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

	adminID := GetUserID(r)
	if adminID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	targetUserID := parts[3]

	// 🚫 запрещаем self-revoke
	if adminID == targetUserID {
		http.Error(w, "cannot revoke admin from yourself", http.StatusBadRequest)
		return
	}

	// 🔁 переключаем admin-флаг
	if err := s.Store.ToggleAdmin(r.Context(), targetUserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 📝 логируем админ-действие
	claims := GetClaims(r)
	_ = s.Store.AddAdminAudit(
		r.Context(),
		adminID,
		"toggle_admin",
		"user",
		targetUserID,
		map[string]any{
			"by_email": claims.Email,
		},
	)

	w.WriteHeader(http.StatusNoContent)
}
