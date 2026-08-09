package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
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
		slog.Error("claims missing from context")
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	userID, err := claims.UserID()
	if err != nil {
		slog.Error("extract user id", "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	peers, err := h.peers.ListByUser(r.Context(), userID)
	if err != nil {
		slog.Error("list peers", "userID", userID, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	resp := make([]map[string]interface{}, 0, len(peers))
	for _, p := range peers {
		resp = append(resp, map[string]interface{}{
			"name":     p.Name,
			"received": p.TrafficReceived,
			"sent":     p.TrafficSent,
		})
	}
	writeJSON(w, 200, resp)
}

func (h *Handler) userInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		slog.Error("claims missing from context")
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	userID, err := claims.UserID()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	user, err := h.repo.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "user not found"})
		} else {
			slog.Error("get user by id", "userID", userID, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	daysLeft := -1
	if user.ExpiresAt != nil {
		until := time.Until(*user.ExpiresAt).Minutes()
		if until < 1 {
			daysLeft = 0
		} else {
			daysLeft = int(math.Ceil(until / (60 * 24)))
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"username":         user.Username,
		"role":             user.Role,
		"expires_at":       user.ExpiresAt,
		"days_left":        daysLeft,
		"subscription_url": "/api/sub/" + user.SubToken + "/mihomo",
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
		slog.Error("claims missing from context")
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	userID, err := claims.UserID()
	if err != nil {
		slog.Error("extract user id", "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	peer, err := h.peers.GetByUserAndName(r.Context(), userID, name)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "config not found"})
		} else {
			slog.Error("get peer", "userID", userID, "name", name, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	status, body, err := h.worker.GetPeerConfig(r.Context(), peer.PeerName)
	if err != nil {
		slog.Error("get peer config from worker", "peer", peer.PeerName, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if status >= 400 {
		slog.Error("get peer config from worker", "peer", peer.PeerName, "status", status, "body", body)
		writeJSON(w, status, body)
		return
	}

	resp := map[string]interface{}{}
	if cfg, ok := body.(worker.PeerConfig); ok {
		resp["config"] = cfg.Config
	}
	resp["received"] = peer.TrafficReceived
	resp["sent"] = peer.TrafficSent
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
		slog.Error("claims missing from context")
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	userID, err := claims.UserID()
	if err != nil {
		slog.Error("extract user id", "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	user, err := h.repo.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "user not found"})
		} else {
			slog.Error("get user by id", "userID", userID, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}
	if user.ExpiresAt != nil && user.ExpiresAt.Before(time.Now()) {
		writeJSON(w, 403, map[string]string{"error": "subscription expired"})
		return
	}

	peers, err := h.peers.ListByUser(r.Context(), userID)
	if err != nil {
		slog.Error("list peer names", "userID", userID, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if len(peers) >= h.config.MaxConfigs {
		writeJSON(w, 400, map[string]string{"error": "config limit reached"})
		return
	}

	peer, err := h.peers.CreatePeer(r.Context(), userID, req.Name)
	if err != nil {
		if errors.Is(err, repo.ErrPeerNameExists) {
			writeJSON(w, 409, map[string]string{"error": "peer name already exists"})
			return
		}
		slog.Error("create peer in db", "userID", userID, "name", req.Name, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	status, body, err := h.worker.CreatePeer(r.Context(), peer.PeerName)
	if err != nil {
		slog.Error("create peer on worker", "peer", peer.PeerName, "err", err)
		_ = h.peers.DeleteByID(r.Context(), peer.ID)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if status != 201 {
		slog.Error("create peer on worker", "peer", peer.PeerName, "status", status, "body", body)
		_ = h.peers.DeleteByID(r.Context(), peer.ID)
		writeJSON(w, status, body)
		return
	}

	writeJSON(w, status, map[string]interface{}{"name": req.Name})
}

func (h *Handler) deleteConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !utils.ValidName(name) {
		writeJSON(w, 400, map[string]string{"error": "name contains invalid characters"})
		return
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		slog.Error("claims missing from context")
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
			slog.Error("get peer", "userID", userID, "name", name, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}
	status, body, err := h.worker.DeletePeer(r.Context(), peer.PeerName)
	if err != nil {
		slog.Error("delete peer on worker", "peer", peer.PeerName, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if status >= 400 {
		slog.Error("delete peer on worker", "peer", peer.PeerName, "status", status, "body", body)
		writeJSON(w, status, body)
		return
	}

	if err := h.peers.DeleteByID(r.Context(), peer.ID); err != nil && !errors.Is(err, repo.ErrNotFound) {
		slog.Error("delete peer in db", "userID", userID, "name", name, "err", err)
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
		slog.Error("claims missing from context")
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	user, err := h.verifyUser(r, claims.Username, req.OldPassword)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			slog.Debug("invalid credentials", "username", claims.Username)
			writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
		} else {
			slog.Error("verify user", "username", claims.Username, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	if err := h.repo.UpdatePassword(r.Context(), user.ID, req.NewPassword); err != nil {
		slog.Error("update password", "userID", user.ID, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "password changed"})
}
