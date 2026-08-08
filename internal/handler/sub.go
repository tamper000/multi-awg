package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tamper000/multi-awg/internal/models"
	"github.com/tamper000/multi-awg/internal/repo"
	"github.com/tamper000/multi-awg/internal/worker"
)

func (h *Handler) loadSub(w http.ResponseWriter, r *http.Request) (*models.User, []models.Peer, worker.Sub, bool) {
	user, err := h.repo.GetBySubToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "subscription not found"})
		} else {
			slog.Error("get user by subscription token", "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return nil, nil, worker.Sub{}, false
	}
	peers, err := h.peers.ListByUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("list subscription peers", "userID", user.ID, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return nil, nil, worker.Sub{}, false
	}
	names := make([]worker.SubPeer, 0, len(peers))
	for _, peer := range peers {
		names = append(names, worker.SubPeer{Name: peer.PeerName, DisplayName: peer.Name})
	}
	status, body, err := h.worker.GetPeersSub(r.Context(), names)
	if err != nil {
		slog.Error("get user subscription from worker", "userID", user.ID, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return nil, nil, worker.Sub{}, false
	}
	if status >= 400 {
		writeJSON(w, status, body)
		return nil, nil, worker.Sub{}, false
	}
	sub, ok := body.(worker.Sub)
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return nil, nil, worker.Sub{}, false
	}
	return user, peers, sub, true
}

func (h *Handler) subConfig(w http.ResponseWriter, r *http.Request) {
	user, _, sub, ok := h.loadSub(w, r)
	if !ok {
		return
	}
	base := "/api/sub/" + user.SubToken
	writeJSON(w, 200, map[string]interface{}{
		"name":        user.Username,
		"mihomo_url":  base + "/mihomo",
		"mihomo_yaml": sub.MihomoYaml,
	})
}

func (h *Handler) subMihomo(w http.ResponseWriter, r *http.Request) {
	_, _, sub, ok := h.loadSub(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sub.MihomoYaml))
}

// subConf keeps direct .conf downloads available for a selected device.
func (h *Handler) subConf(w http.ResponseWriter, r *http.Request) {
	user, peers, _, ok := h.loadSub(w, r)
	if !ok {
		return
	}
	name := chi.URLParam(r, "name")
	if name == "" {
		name = r.URL.Query().Get("name")
	}
	if name == "" {
		writeJSON(w, 400, map[string]string{"error": "config name required"})
		return
	}
	var peer *models.Peer
	for i := range peers {
		if peers[i].Name == name {
			peer = &peers[i]
			break
		}
	}
	if peer == nil || peer.UserID != user.ID {
		writeJSON(w, 404, map[string]string{"error": "config not found"})
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
	cfg, ok := body.(worker.PeerConfig)
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", `attachment; filename="`+peer.Name+`.conf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(cfg.Config))
}
