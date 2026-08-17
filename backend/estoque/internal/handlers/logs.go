package handlers

import (
	"net/http"
	"strconv"

	"korp/estoque/internal/repository"
)

type LogsHandler struct {
	Repo *repository.ActivityRepository
}

func (h *LogsHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	logs, err := h.Repo.List(r.Context(), limit)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}