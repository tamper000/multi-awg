package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tamper000/multi-awg/internal/auth"
	"github.com/tamper000/multi-awg/internal/repo"
)

type ctxKey string

const claimsKey ctxKey = "claims"

func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*auth.Claims)
	return claims, ok
}

// authMiddleware проверяет JWT (Bearer) и кладёт claims в context.
func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}

		claims, err := h.tokens.Parse(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			slog.Debug("parse token", "err", err)
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}

		exists, err := h.sessions.SessionExists(r.Context(), claims.JTI)
		if err != nil {
			slog.Error("session exists", "jti", claims.JTI, "err", err)
			writeJSON(w, 500, map[string]string{"error": "internal error"})
			return
		}
		if !exists {
			slog.Debug("session not found", "jti", claims.JTI)
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminMiddleware поверх authMiddleware: только role=admin.
func (h *Handler) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims == nil || claims.Role != repo.RoleAdmin {
			slog.Debug("admin access denied")
			writeJSON(w, 403, map[string]string{"error": "forbidden"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
