package handler

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tamper000/multi-awg/internal/repo"
	"github.com/tamper000/multi-awg/internal/utils"
	"github.com/tamper000/multi-awg/internal/worker"
)

func (h *Handler) listConfigs(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	userID, err := claims.UserID()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	peers, err := h.peers.ListByUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	statsByName := map[string]worker.Stats{}
	if status, body, err := h.worker.GetStats(r.Context()); err == nil && status < 400 {
		if stats, ok := body.([]worker.Stats); ok {
			for _, s := range stats {
				statsByName[s.Name] = s
			}
		}
	}

	resp := make([]map[string]interface{}, 0, len(peers))
	for _, p := range peers {
		item := map[string]interface{}{"name": p.Name, "sub_token": p.SubToken}
		if s, ok := statsByName[p.PeerName]; ok {
			item["received"] = s.Received
			item["sent"] = s.Sent
		}
		resp = append(resp, item)
	}
	writeJSON(w, 200, resp)
}

func (h *Handler) userInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	user, err := h.repo.GetByUsername(r.Context(), claims.Username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "user not found"})
		} else {
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	daysLeft := -1
	if user.ExpiresAt != nil {
		daysLeft = int(math.Ceil(time.Until(*user.ExpiresAt).Hours() / 24))
		if daysLeft < 0 {
			daysLeft = 0
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"username":   user.Username,
		"role":       user.Role,
		"expires_at": user.ExpiresAt,
		"days_left":  daysLeft,
	})
}

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !utils.ValidName(name) {
		writeJSON(w, 400, map[string]string{"error": "name contains invalid characters"})
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	userID, err := claims.UserID()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	peer, err := h.peers.GetByUserAndName(r.Context(), userID, name)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "config not found"})
		} else {
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	status, body, err := h.worker.GetPeerConfig(r.Context(), peer.PeerName)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if status >= 400 {
		writeJSON(w, status, body)
		return
	}

	resp := map[string]interface{}{}
	if cfg, ok := body.(worker.PeerConfig); ok {
		resp["config"] = cfg.Config
	}
	if status, body, err := h.worker.GetPeerStats(r.Context(), peer.PeerName); err == nil && status < 400 {
		if st, ok := body.(worker.Stats); ok {
			resp["received"] = st.Received
			resp["sent"] = st.Sent
		}
	}
	writeJSON(w, 200, resp)
}

func (h *Handler) createConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name required"})
		return
	}
	if !utils.ValidName(req.Name) {
		writeJSON(w, 400, map[string]string{"error": "name contains invalid characters"})
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	userID, err := claims.UserID()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	user, err := h.repo.GetByUsername(r.Context(), claims.Username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "user not found"})
		} else {
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}
	if user.ExpiresAt != nil && user.ExpiresAt.Before(time.Now()) {
		writeJSON(w, 403, map[string]string{"error": "subscription expired"})
		return
	}

	names, err := h.peers.ListNamesByUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if len(names) >= h.config.MaxConfigs {
		writeJSON(w, 400, map[string]string{"error": "config limit reached"})
		return
	}

	peerName := claims.Username + "." + req.Name
	status, body, err := h.worker.CreatePeer(r.Context(), peerName)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if status != 201 {
		writeJSON(w, status, body)
		return
	}

	subToken, err := h.peers.CreatePeer(r.Context(), userID, req.Name, peerName)
	if err != nil {
		_, _, _ = h.worker.DeletePeer(r.Context(), peerName)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, status, map[string]interface{}{"name": req.Name, "sub_token": subToken})
}

func (h *Handler) deleteConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !utils.ValidName(name) {
		writeJSON(w, 400, map[string]string{"error": "name contains invalid characters"})
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	peerName := claims.Username + "." + name
	status, body, err := h.worker.DeletePeer(r.Context(), peerName)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if status >= 400 {
		writeJSON(w, status, body)
		return
	}

	userID, err := claims.UserID()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if err := h.peers.DeleteByName(r.Context(), userID, name); err != nil && !errors.Is(err, repo.ErrNotFound) {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, status, body)
}

func (h *Handler) password(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old"`
		NewPassword string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		writeJSON(w, 400, map[string]string{"error": "old and new password required"})
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	user, err := h.verifyUser(r, claims.Username, req.OldPassword)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
		} else {
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	if err := h.repo.UpdatePassword(r.Context(), user.ID, req.NewPassword); err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "password changed"})
}
