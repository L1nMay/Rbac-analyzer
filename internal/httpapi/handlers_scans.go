package httpapi

import (
	"encoding/json"
	"net/http"

	"rbac-analyzer/internal/store"
)

// POST /api/app/scans/diff
type diffReq struct {
	BaseID   string `json:"baseId"`
	TargetID string `json:"targetId"`
}

func (s *Server) handleDiffScans(w http.ResponseWriter, r *http.Request) {
	var req diffReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad json"})
		return
	}

	baseSum, baseFull, err := s.Store.GetScanReport(r.Context(), req.BaseID)
	if err != nil {
		if store.IsNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "base scan not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	targetSum, targetFull, err := s.Store.GetScanReport(r.Context(), req.TargetID)
	if err != nil {
		if store.IsNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "target scan not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"base": map[string]any{
			"summary": baseSum,
			"report":  baseFull,
		},
		"target": map[string]any{
			"summary": targetSum,
			"report":  targetFull,
		},
	})
}
