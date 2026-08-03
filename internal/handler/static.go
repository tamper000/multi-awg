package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (h *Handler) serveSPA(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	rel := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(h.config.StaticDir, rel)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	http.ServeFile(w, r, filepath.Join(h.config.StaticDir, "index.html"))
}
