package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/tamper000/multi-awg/internal/auth"
)

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Username == "" || req.Password == "" {
		writeJSON(w, 400, map[string]string{"error": "username and password required"})
		return
	}

	user, err := h.verifyUser(r, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
		} else {
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	jti, err := auth.NewJTI()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	expiresAt := time.Now().Add(h.config.TokenTTL)
	if err := h.sessions.CreateSession(r.Context(), user.ID, jti, expiresAt); err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	token, _, err := h.tokens.Generate(user.ID, user.Username, user.Role, jti)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"token":                token,
		"user":                 user,
		"subscription_expired": user.ExpiresAt != nil && user.ExpiresAt.Before(time.Now()),
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		if claims, err := h.tokens.Parse(strings.TrimPrefix(header, "Bearer ")); err == nil {
			_ = h.sessions.DeleteSession(r.Context(), claims.JTI)
		}
	}
	writeJSON(w, 200, map[string]string{"status": "logged out"})
}
