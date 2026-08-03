package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tamper000/multi-awg/internal/models"
	"github.com/tamper000/multi-awg/internal/repo"
	"github.com/tamper000/multi-awg/internal/worker"
)

func (h *Handler) loadSub(w http.ResponseWriter, r *http.Request) (*models.Peer, worker.Sub, bool) {
	token := chi.URLParam(r, "token")
	peer, err := h.peers.GetBySubToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "subscription not found"})
		} else {
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return nil, worker.Sub{}, false
	}

	status, body, err := h.worker.GetPeerSub(r.Context(), peer.PeerName)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return nil, worker.Sub{}, false
	}
	if status >= 400 {
		writeJSON(w, status, body)
		return nil, worker.Sub{}, false
	}
	sub, ok := body.(worker.Sub)
	if !ok {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return nil, worker.Sub{}, false
	}
	return peer, sub, true
}

func (h *Handler) subConfig(w http.ResponseWriter, r *http.Request) {
	peer, sub, ok := h.loadSub(w, r)
	if !ok {
		return
	}

	base := "https://" + r.Host + "/api/sub/" + chi.URLParam(r, "token")
	writeJSON(w, 200, map[string]interface{}{
		"name":           peer.Name,
		"conf":           sub.Conf,
		"vpn_link":       sub.VPNLink,
		"sub_mihomo_url": base + "/mihomo",
	})
}

func (h *Handler) subMihomo(w http.ResponseWriter, r *http.Request) {
	_, sub, ok := h.loadSub(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(200)
	_, _ = w.Write([]byte(sub.MihomoYaml))
}

func (h *Handler) subConf(w http.ResponseWriter, r *http.Request) {
	peer, sub, ok := h.loadSub(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", `attachment; filename="`+peer.Name+`.conf"`)
	w.WriteHeader(200)
	_, _ = w.Write([]byte(sub.Conf))
}
