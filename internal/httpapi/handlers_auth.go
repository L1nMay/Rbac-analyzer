package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"rbac-analyzer/internal/security"
)

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	OrgName  string `json:"orgName"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResp struct {
	Token string `json:"token"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "email required and password >= 8", http.StatusBadRequest)
		return
	}
	if req.OrgName == "" {
		req.OrgName = "My Organization"
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "hash error", http.StatusInternalServerError)
		return
	}

	u, err := s.Store.CreateUser(r.Context(), req.Email, hash)
	if err != nil {
		http.Error(w, "create user failed (maybe already exists): "+err.Error(), http.StatusBadRequest)
		return
	}

	_, err = s.Store.CreateOrgForOwner(r.Context(), u.ID, req.OrgName)
	if err != nil {
		http.Error(w, "create org failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	token, _ := security.SignJWT([]byte(s.Cfg.JWTSecret), security.Claims{
		Sub:   u.ID,
		Email: u.Email,
		Exp:   time.Now().Add(7 * 24 * time.Hour).Unix(),
	})

	writeJSON(w, authResp{Token: token})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}

	u, err := s.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if !security.CheckPassword(u.PasswordHash, req.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, _ := security.SignJWT([]byte(s.Cfg.JWTSecret), security.Claims{
		Sub:   u.ID,
		Email: u.Email,
		Exp:   time.Now().Add(7 * 24 * time.Hour).Unix(),
	})

	writeJSON(w, authResp{Token: token})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
