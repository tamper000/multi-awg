package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tamper000/multi-awg/internal/models"
	"github.com/tamper000/multi-awg/internal/repo"
	"github.com/tamper000/multi-awg/internal/utils/password"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

var errInvalidCredentials = errors.New("invalid credentials")

func (h *Handler) verifyUser(r *http.Request, username, plainPassword string) (*models.User, error) {
	user, err := h.repo.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, errInvalidCredentials
		}
		return nil, err
	}
	if !password.Verify(user.PasswordHash, plainPassword) {
		return nil, errInvalidCredentials
	}
	return user, nil
}
