package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tamper000/multi-awg/internal/repo"
	"github.com/tamper000/multi-awg/internal/utils"
	"github.com/tamper000/multi-awg/internal/utils/password"
	"github.com/tamper000/multi-awg/internal/worker"
)

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.ListUsers(r.Context())
	if err != nil {
		slog.Error("list users", "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, 200, users)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Days     int    `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Username == "" || req.Days <= 0 {
		writeJSON(w, 400, map[string]string{"error": "username and positive days required"})
		return
	}
	if !utils.ValidName(req.Username) {
		writeJSON(w, 400, map[string]string{"error": "username contains invalid characters"})
		return
	}

	if _, err := h.repo.GetByUsername(r.Context(), req.Username); err == nil {
		writeJSON(w, 409, map[string]string{"error": "username already exists"})
		return
	} else if !errors.Is(err, repo.ErrNotFound) {
		slog.Error("get user by username", "username", req.Username, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	plain, err := password.Generate(16)
	if err != nil {
		slog.Error("generate password", "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	expiresAt := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour)
	if err := h.repo.CreateUser(r.Context(), req.Username, plain, repo.RoleUser, &expiresAt); err != nil {
		slog.Error("create user", "username", req.Username, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, 201, map[string]interface{}{
		"username":   req.Username,
		"password":   plain,
		"expires_at": expiresAt,
	})
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	user, err := h.repo.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "user not found"})
		} else {
			slog.Error("get user by username", "username", username, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	peers, err := h.peers.ListByUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("list peers", "userID", user.ID, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	statsByName := map[string]worker.Stats{}
	status, body, err := h.worker.GetStats(r.Context())
	if err != nil {
		slog.Error("get stats from worker", "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if status >= 400 {
		slog.Error("get stats from worker", "status", status, "body", body)
		writeJSON(w, status, body)
		return
	}
	if stats, ok := body.([]worker.Stats); ok {
		for _, s := range stats {
			statsByName[s.Name] = s
		}
	}

	configs := make([]map[string]interface{}, 0, len(peers))
	for _, p := range peers {
		cfg := map[string]interface{}{"name": p.Name, "sub_token": p.SubToken}
		if s, ok := statsByName[p.PeerName]; ok {
			cfg["ip"] = s.IP
			cfg["received"] = s.Received
			cfg["sent"] = s.Sent
			cfg["last_handshake"] = s.LastHandshake
		}
		configs = append(configs, cfg)
	}

	writeJSON(w, 200, map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"expires_at": user.ExpiresAt,
		"created_at": user.CreatedAt,
		"configs":    configs,
	})
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	user, err := h.repo.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "user not found"})
		} else {
			slog.Error("get user by username", "username", username, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	names, err := h.peers.ListNamesByUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("list peer names", "userID", user.ID, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if len(names) > 0 {
		status, body, err := h.worker.DeletePeers(r.Context(), names)
		if err != nil {
			slog.Error("delete peers on worker", "names", names, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
			return
		}
		if status >= 400 {
			slog.Error("delete peers on worker", "status", status, "body", body)
			writeJSON(w, status, body)
			return
		}
	}

	if err := h.repo.DeleteUser(r.Context(), username); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "user not found"})
		} else {
			slog.Error("delete user", "username", username, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (h *Handler) patchUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var req struct {
		Days int `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Days <= 0 {
		writeJSON(w, 400, map[string]string{"error": "positive days required"})
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour)
	if err := h.repo.UpdateExpiry(r.Context(), username, &expiresAt); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, 404, map[string]string{"error": "user not found"})
		} else {
			slog.Error("update expiry", "username", username, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
		}
		return
	}

	user, err := h.repo.GetByUsername(r.Context(), username)
	if err != nil {
		slog.Error("get user by username", "username", username, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	names, err := h.peers.ListNamesByUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("list peer names", "userID", user.ID, "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	status, _, err := h.worker.UnfreezePeers(r.Context(), names)
	if err != nil {
		slog.Error("unfreeze peers", "names", names, "err", err)
	} else if status < 400 {
		if err := h.repo.SetFrozen(r.Context(), user.ID, false); err != nil {
			slog.Error("set unfrozen", "userID", user.ID, "err", err)
		}
	}
	writeJSON(w, 200, map[string]interface{}{
		"username":   username,
		"expires_at": expiresAt,
	})
}

func (h *Handler) syncWorker(w http.ResponseWriter, r *http.Request) {
	status, body, err := h.worker.Sync(r.Context())
	if err != nil {
		slog.Error("sync worker", "err", err)
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	if status >= 400 {
		slog.Error("sync worker", "status", status, "body", body)
		writeJSON(w, status, body)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "synced"})
}
